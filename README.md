# Go Runner

`Go Runner` — это пакет для управления запуском и остановкой приложений в Go с поддержкой graceful shutdown. Он позволяет регистрировать приложения, запускать их параллельно и корректно завершать работу при получении сигналов завершения (например, `SIGTERM` или `SIGINT`).

---

## Основные возможности

- **Параллельный запуск приложений**: Приложения запускаются в отдельных горутинах, что позволяет им работать одновременно.
- **Graceful Shutdown**: При получении сигнала завершения (например, `SIGTERM` или `SIGINT`) все запущенные приложения корректно останавливаются.
- **Регистрация приложений**: Приложения регистрируются с помощью интерфейса `app`, который требует реализации методов `Start()` и `Stop()`.
- **Shutdown Hooks**: Возможность регистрации функций, которые будут вызваны при остановке приложения.
- **Логирование**: Поддержка логгирования через интерфейс `Logger`.

---

## Установка

Для использования пакета добавьте его в ваш проект:

```bash
go get github.com/aatumaykin/go-runner
```

## Использование

### 1. Регистрация приложений

**Приложения должны реализовать интерфейс app:**

```go
type app interface {
    Start() error
    Stop() error
}
```

**Пример реализации приложения:**

```go
type MyApp struct {
    Name string
}

func (a *MyApp) Start() error {
    fmt.Println("Starting", a.Name)
    return nil
}

func (a *MyApp) Stop() error {
    fmt.Println("Stopping", a.Name)
    return nil
}
```

### 2. Создание и запуск AppsRunner

```go
package main

import (
    "context"
    "log"

    "github.com/yourusername/go-runner"
)

func main() {
    // Создаем логгер (реализация интерфейса Logger)
    logger := &MyLogger{}

    // Создаем AppsRunner
    runner := go_runner.New(logger)

    // Регистрируем приложения
    app1 := &MyApp{Name: "App1"}
    app2 := &MyApp{Name: "App2"}
    runner.RegisterApp(app1)
    runner.RegisterApp(app2)

    // Регистрируем shutdown hook
    runner.RegisterShutdownHook(func() error {
        log.Println("Shutdown hook executed")
        return nil
    })

    // Запускаем приложения
    if err := runner.Run(context.Background()); err != nil {
        log.Fatal("Failed to run apps:", err)
    }
}
```

### Интерфейс Logger

Пакет использует интерфейс Logger для логгирования. Вы можете реализовать свой логгер или использовать любой совместимый логгер (например, logrus, zap и т.д.).

**Пример интерфейса:**

```go
type Logger interface {
    Debug(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Error(msg string, args ...interface{})
}
```

**Пример логгера**

```go
type MyLogger struct{}

func (l *MyLogger) Debug(msg string, args ...interface{}) {
    log.Printf("[DEBUG] "+msg, args...)
}

func (l *MyLogger) Info(msg string, args ...interface{}) {
    log.Printf("[INFO] "+msg, args...)
}

func (l *MyLogger) Error(msg string, args ...interface{}) {
    log.Printf("[ERROR] "+msg, args...)
}
```

### Graceful Shutdown

При получении сигналов SIGTERM или SIGINT пакет корректно останавливает все запущенные приложения в порядке, обратном их запуску. Также вызываются зарегистрированные shutdown hooks.

### Обработка ошибок

Если приложение завершается с ошибкой, все остальные приложения также останавливаются.

Ошибки при остановке приложений логируются, но не прерывают процесс остановки.