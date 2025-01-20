package go_runner

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

type (
	callback func() error

	appStruct struct {
		Name  string
		Start callback
		Stop  callback
	}

	// app интерфейс
	app interface {
		Start() error
		Stop() error
	}

	// Runner сервис для запуска приложений в режиме graceful shutdown
	Runner struct {
		apps   []appStruct
		logger Logger
	}
)

// New создает новый экземпляр Runner с указанным логгером.
func New(logger Logger) *Runner {
	return &Runner{
		apps:   make([]appStruct, 0),
		logger: logger,
	}
}

// RegisterApp регистрирует приложение, реализующее интерфейс app.
func (r *Runner) RegisterApp(instance app) {
	r.RegisterNamedApp("", instance)
}

// RegisterNamedApp регистрирует приложение с указанным именем.
func (r *Runner) RegisterNamedApp(name string, instance app) {
	r.apps = append(r.apps, appStruct{
		Name:  name,
		Start: instance.Start,
		Stop:  instance.Stop,
	})
}

// RegisterShutdownHook регистрирует функцию, которая будет вызвана при остановке приложения.
func (r *Runner) RegisterShutdownHook(stop callback) {
	if stop == nil {
		return
	}

	r.apps = append(r.apps, appStruct{
		Start: nil,
		Stop:  stop,
	})
}

func (r *Runner) Run(ctx context.Context) error {
	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Создаем errgroup с привязкой к контексту
	eg, ctx := errgroup.WithContext(ctx)

	// Флаги для отслеживания запущенных приложений
	started := make([]bool, len(r.apps))

	// Запускаем все приложения
	for i, a := range r.apps {
		if a.Start == nil {
			continue
		}

		r.logger.Debug("start application", "app", a.Name)

		// Запускаем приложение в отдельной горутине
		eg.Go(func() error {
			err := a.Start()
			if err != nil {
				r.logger.Debug("application finished", "app", a.Name, "error", err)
				cancel() // Отменяем контекст при ошибке
				return err
			}

			// Помечаем приложение как запущенное только в случае успеха
			started[i] = true
			r.logger.Debug("application started", "app", a.Name)
			return nil
		})
	}

	// Graceful shutdown
	eg.Go(func() error {
		<-ctx.Done()

		var err error
		// Останавливаем только запущенные приложения
		for i, a := range r.apps {
			if a.Stop == nil || !started[i] {
				continue
			}

			r.logger.Debug("stop application", "app", a.Name)
			if stopErr := a.Stop(); stopErr != nil {
				r.logger.Error("application stop error", "app", a.Name, "error", stopErr)
				err = stopErr
			}
		}

		// Вызываем shutdown hook
		for _, a := range r.apps {
			if a.Start == nil && a.Stop != nil { // Это shutdown hook
				r.logger.Debug("calling shutdown hook", "app", a.Name)
				if hookErr := a.Stop(); hookErr != nil {
					r.logger.Error("shutdown hook error", "app", a.Name, "error", hookErr)
					err = hookErr
				}
			}
		}

		return err
	})

	// Обработка сигнала завершения
	eg.Go(func() error {
		sig := []os.Signal{syscall.SIGTERM, syscall.SIGINT}
		ch := make(chan os.Signal, len(sig))
		signal.Notify(ch, sig...)

		select {
		case <-ch:
			cancel()
			return ErrInterruptedBySignal
		case <-ctx.Done():
			return nil
		}
	})

	if err := eg.Wait(); err != nil {
		if errors.Is(err, ErrInterruptedBySignal) {
			r.logger.Debug("shutting down by signal")
		} else {
			r.logger.Error("terminating with error", "error", err)
			return err
		}
	}

	r.logger.Info("application was stopped")
	return nil
}
