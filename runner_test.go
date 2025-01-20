package go_runner

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockLogger — мок логгера для тестов
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, args ...any) {
	m.Called(append([]any{msg}, args...)...)
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.Called(append([]any{msg}, args...)...)
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.Called(append([]any{msg}, args...)...)
}

func (m *MockLogger) Warn(msg string, args ...any) {
	m.Called(append([]any{msg}, args...)...)
}

// MockApp — мок приложения для тестов
type MockApp struct {
	mock.Mock
}

func (m *MockApp) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockApp) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func TestAppsRunner_Run_Success(t *testing.T) {
	loggerMock := &MockLogger{}
	appMock := &MockApp{}

	// Ожидаем успешный запуск и остановку
	appMock.On("Start").Return(nil)
	appMock.On("Stop").Return(nil)

	// Ожидаем вызов Debug с тремя аргументами
	loggerMock.On("Debug", "start application", "app", "").Once()
	loggerMock.On("Debug", "application started", "app", "").Once()
	loggerMock.On("Debug", "stop application", "app", "").Once()
	loggerMock.On("Info", "application was stopped").Once()

	runner := New(loggerMock)
	runner.RegisterApp(appMock)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runner.Run(ctx)
	require.NoError(t, err)

	appMock.AssertExpectations(t)
	loggerMock.AssertExpectations(t)
}

func TestAppsRunner_Run_StartError(t *testing.T) {
	loggerMock := &MockLogger{}
	appMock := &MockApp{}

	// Ожидаем ошибку при запуске
	expectedErr := errors.New("start error")
	appMock.On("Start").Return(expectedErr)

	// Ожидаем вызов Debug для запуска и завершения приложения
	loggerMock.On("Debug", "start application", "app", "").Once()
	loggerMock.On("Debug", "application finished", "app", "", "error", expectedErr).Once()
	loggerMock.On("Error", "terminating with error", "error", expectedErr).Once()

	runner := New(loggerMock)
	runner.RegisterApp(appMock)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runner.Run(ctx)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)

	appMock.AssertExpectations(t)
	loggerMock.AssertExpectations(t)
}

func TestAppsRunner_Run_StopError(t *testing.T) {
	loggerMock := &MockLogger{}
	appMock := &MockApp{}

	// Ожидаем ошибку при остановке
	expectedErr := errors.New("stop error")

	// Ожидаем успешный запуск, но ошибку при остановке
	appMock.On("Start").Return(nil)
	appMock.On("Stop").Return(expectedErr)

	loggerMock.On("Debug", "start application", "app", "").Once()
	loggerMock.On("Debug", "application started", "app", "").Once()
	loggerMock.On("Debug", "stop application", "app", "").Once()
	loggerMock.On("Error", "application stop error", "app", "", "error", expectedErr).Once()
	loggerMock.On("Error", "terminating with error", "error", expectedErr).Once()

	runner := New(loggerMock)
	runner.RegisterApp(appMock)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runner.Run(ctx)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)

	appMock.AssertExpectations(t)
	loggerMock.AssertExpectations(t)
}

func TestAppsRunner_Run_ShutdownBySignal(t *testing.T) {
	loggerMock := &MockLogger{}
	appMock := &MockApp{}

	// Ожидаем успешный запуск и остановку по сигналу
	appMock.On("Start").Return(nil)
	appMock.On("Stop").Return(nil)

	loggerMock.On("Debug", "start application", "app", "").Once()
	loggerMock.On("Debug", "application started", "app", "").Once()
	loggerMock.On("Debug", "stop application", "app", "").Once()
	loggerMock.On("Debug", "shutting down by signal").Once()
	loggerMock.On("Info", "application was stopped").Once()

	runner := New(loggerMock)
	runner.RegisterApp(appMock)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Эмулируем отправку сигнала SIGTERM
	go func() {
		time.Sleep(100 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(syscall.SIGTERM)
	}()

	err := runner.Run(ctx)
	require.NoError(t, err)

	appMock.AssertExpectations(t)
	loggerMock.AssertExpectations(t)
}

func TestAppsRunner_RegisterShutdownHook(t *testing.T) {
	loggerMock := &MockLogger{}
	appMock := &MockApp{}

	// Ожидаем успешный запуск и остановку
	appMock.On("Start").Return(nil)
	appMock.On("Stop").Return(nil)

	shutdownHookCalled := false
	shutdownHook := func() error {
		shutdownHookCalled = true
		return nil
	}

	loggerMock.On("Debug", "start application", "app", "").Once()
	loggerMock.On("Debug", "application started", "app", "").Once()
	loggerMock.On("Debug", "stop application", "app", "").Once()
	loggerMock.On("Debug", "calling shutdown hook", "app", "").Once()
	loggerMock.On("Info", "application was stopped").Once()

	runner := New(loggerMock)
	runner.RegisterApp(appMock)
	runner.RegisterShutdownHook(shutdownHook)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := runner.Run(ctx)
	require.NoError(t, err)
	assert.True(t, shutdownHookCalled, "shutdown hook should be called")

	appMock.AssertExpectations(t)
	loggerMock.AssertExpectations(t)
}
