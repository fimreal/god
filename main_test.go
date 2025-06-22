package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrefixedWriter(t *testing.T) {
	var buf bytes.Buffer
	pw := &prefixedWriter{
		name:   "test",
		writer: &buf,
	}

	// Test single line
	testData := []byte("Hello World")
	n, err := pw.Write(testData)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected %d bytes written, got %d", len(testData), n)
	}
	expected := "[test] Hello World"
	if !strings.Contains(buf.String(), expected) {
		t.Errorf("Expected output to contain %q, got %q", expected, buf.String())
	}

	// Test empty input
	buf.Reset()
	n, err = pw.Write([]byte{})
	if err != nil {
		t.Errorf("Write empty failed: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected 0 bytes written for empty input, got %d", n)
	}
}

func TestHealthCheckHandler(t *testing.T) {
	mgr := NewManager(false)

	// Test empty processes - should return 200 since no init tasks are running
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mgr.HealthCheckHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for empty processes, got %d", w.Code)
	}

	// Test with healthy services (no init tasks)
	mgr.processes = []*Process{
		{Name: "app1", Alive: true, Type: TaskTypeService},
		{Name: "app2", Alive: true, Type: TaskTypeService},
	}

	// Signal that init is done
	close(mgr.initDone)

	w = httptest.NewRecorder()
	mgr.HealthCheckHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for healthy services, got %d", w.Code)
	}

	// Test with unhealthy services
	mgr.processes[1].Alive = false

	w = httptest.NewRecorder()
	mgr.HealthCheckHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for unhealthy services, got %d", w.Code)
	}

	// Test with failed init task
	mgr2 := NewManager(false)
	mgr2.processes = []*Process{
		{Name: "init1", Success: false, Type: TaskTypeInit},
	}
	close(mgr2.initDone)

	w = httptest.NewRecorder()
	mgr2.HealthCheckHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for failed init task, got %d", w.Code)
	}
}

func TestStringSlice(t *testing.T) {
	var ss StringSlice

	// Test Set method
	err := ss.Set("test1")
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}
	err = ss.Set("test2")
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// Test String method
	result := ss.String()
	expected := "test1, test2"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestNewManager(t *testing.T) {
	mgr := NewManager(false)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if len(mgr.processes) != 0 {
		t.Error("New manager should have empty processes slice")
	}
}
