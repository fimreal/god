// writer.go
// 日志输出前缀工具，负责为每行输出加上进程名前缀。
package main

import (
	"fmt"
	"io"
	"strings"
)

// createPrefixedWriter creates a writer that prefixes each line with the process name
func createPrefixedWriter(name string, writer io.Writer) io.Writer {
	return &prefixedWriter{
		name:   name,
		writer: writer,
	}
}

type prefixedWriter struct {
	name   string
	writer io.Writer
}

func (pw *prefixedWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	lines := strings.Split(string(p), "\n")
	totalWritten := 0
	for _, line := range lines {
		if line != "" {
			prefixedLine := fmt.Sprintf("[%s] %s\n", pw.name, line)
			written, err := pw.writer.Write([]byte(prefixedLine))
			if err != nil {
				return totalWritten, err
			}
			totalWritten += written
		}
	}
	return len(p), nil
}
