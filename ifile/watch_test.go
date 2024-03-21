package ifile

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWatch(t *testing.T) {
	os.Remove(testIfile)
	os.Remove("test_txtfile")

	j := NewWatchJob(testIfile, ModeSyncthing, nil, nil, zap.NewNop())

	var (
		walk      = j.walk
		content   []byte
		walkCount = 0
		mu        sync.Mutex
	)

	j.walk = func() error {
		walkErr := walk()
		c, err := os.ReadFile(testIfile)

		mu.Lock()
		content = c
		walkCount++
		mu.Unlock()

		if !os.IsNotExist(err) {
			require.NoError(t, err)
		}
		return walkErr
	}

	go func() {
		err := j.Run()
		require.NoError(t, err)
	}()

	for {
		if j.Status() == WatchJobStatusRunning {
			break
		}
		time.Sleep(time.Millisecond * 50)
	}

	err := os.WriteFile(testIfile, []byte(""), 0644)
	require.NoError(t, err)

	for {
		mu.Lock()
		if walkCount >= 1 {
			mu.Unlock()
			break
		}
		mu.Unlock()
		time.Sleep(time.Millisecond * 50)
	}

	for _, line := range strings.Split(string(content), "\n") {
		entry := "/ifile/test_txtfile"
		if line == entry {
			t.Fatalf("this shouldn't have been in ifile: %s", entry)
		}
	}

	err = j.Shutdown()
	require.NoError(t, err)
}

// Make sure newly created .gitignore and ignored file gets watched and entry gets added.
func TestWatchIgnore(t *testing.T) {
	os.Remove(testIfile)
	os.Remove("test_txtfile")
	os.Remove(".gitignore")

	defer os.Remove("test_txtfile")
	defer os.Remove(".gitignore")

	j := NewWatchJob(testIfile, ModeSyncthing, nil, nil, zap.NewNop())

	var (
		walk      = j.walk
		content   []byte
		walkCount = 0
		mu        sync.Mutex
	)

	j.walk = func() error {
		walkErr := walk()
		c, err := os.ReadFile(testIfile)

		mu.Lock()
		content = c
		walkCount++
		mu.Unlock()

		if !os.IsNotExist(err) {
			require.NoError(t, err)
		}
		return walkErr
	}
	go func() {
		err := j.Run()
		require.NoError(t, err)
	}()

	for {
		if j.Status() == WatchJobStatusRunning {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	err := os.WriteFile(".gitignore", []byte("test_txtfile"), 0644)
	require.NoError(t, err)

	err = os.WriteFile("test_txtfile", []byte(""), 0644)
	require.NoError(t, err)

	for {
		mu.Lock()
		if walkCount >= 2 {
			mu.Unlock()
			break
		}
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}

	found := false
	mu.Lock()
	for _, line := range strings.Split(string(content), "\n") {
		entry := "/ifile/test_txtfile"
		if line == entry {
			found = true
		}
	}
	mu.Unlock()
	require.True(t, found)

	err = j.Shutdown()
	require.NoError(t, err)
}

// Make sure newly created .gitignore and ignored file inside newly created directory gets watched and entry gets added.
func TestWatchIgnoreNewlyCreatedDir(t *testing.T) {
	os.Remove(testIfile)
	os.RemoveAll("testdir")
	os.Remove(".gitignore")

	defer os.RemoveAll("testdir")
	defer os.Remove(".gitignore")

	j := NewWatchJob(testIfile, ModeSyncthing, nil, nil, zap.NewNop())

	var (
		walk      = j.walk
		content   []byte
		walkCount = 0
		mu        sync.Mutex
	)

	j.walk = func() error {
		walkErr := walk()
		c, err := os.ReadFile(testIfile)

		mu.Lock()
		content = c
		walkCount++
		mu.Unlock()

		if !os.IsNotExist(err) {
			require.NoError(t, err)
		}
		return walkErr
	}
	go func() {
		err := j.Run()
		require.NoError(t, err)
	}()

	for {
		if j.Status() == WatchJobStatusRunning {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	err := os.Mkdir("testdir", 0755)
	require.NoError(t, err)

	err = os.WriteFile(".gitignore", []byte("/testdir/test_txtfile"), 0644)
	require.NoError(t, err)

	err = os.WriteFile("testdir/test_txtfile", []byte(""), 0644)
	require.NoError(t, err)

	for {
		mu.Lock()
		if walkCount >= 4 {
			mu.Unlock()
			break
		}
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}

	found := false
	mu.Lock()
	for _, line := range strings.Split(string(content), "\n") {
		entry := "/ifile/testdir/test_txtfile"
		if line == entry {
			found = true
		}
	}
	mu.Unlock()
	require.True(t, found)

	err = j.Shutdown()
	require.NoError(t, err)
}

func TestWatchFail(t *testing.T) {
	os.Remove(testIfile)
	os.Remove("test_txtfile")

	j := NewWatchJob(testIfile, ModeSyncthing, nil, nil, zap.NewNop())
	j.failAfter = 4

	var (
		walkCount = 0
		mu        sync.Mutex
	)

	j.walk = func() error {
		mu.Lock()
		walkCount++
		walkCount := walkCount - 1
		mu.Unlock()

		if walkCount == 0 {
			return nil
		}
		return fmt.Errorf("test walk error")
	}

	go func() {
		err := j.Run()
		require.Error(t, err)
	}()

	for {
		if j.Status() == WatchJobStatusRunning {
			break
		}
		time.Sleep(time.Millisecond * 50)
	}

	for {
		mu.Lock()
		if walkCount >= 1 {
			mu.Unlock()
			break
		}
		mu.Unlock()
		time.Sleep(time.Millisecond * 50)
	}

	// Trigger walk
	err := os.WriteFile("test_txtfile", nil, 0644)
	require.NoError(t, err)
	err = os.Remove("test_txtfile")
	require.NoError(t, err)

	for {
		mu.Lock()
		if walkCount >= 2 {
			mu.Unlock()
			break
		}
		mu.Unlock()
		time.Sleep(time.Millisecond * 50)
	}

	info := j.Info()
	require.Greater(t, len(info.Errors), 0)

	// Ensure 4 seconds (value of failAfter) has passed.
	time.Sleep(4010 * time.Millisecond)

	// Trigger walk again
	err = os.WriteFile("test_txtfile2", nil, 0644)
	require.NoError(t, err)
	err = os.Remove("test_txtfile2")
	require.NoError(t, err)

	for {
		if j.Status() == WatchJobStatusFailed {
			break
		}
		time.Sleep(time.Millisecond * 50)
	}

	info = j.Info()
	require.Greater(t, len(info.Errors), 0)

	err = j.Shutdown()
	require.NoError(t, err)
}

func TestWatchFailImmediately(t *testing.T) {
	os.Remove(testIfile)
	os.Remove("test_txtfile")

	runHooks := func() error { return fmt.Errorf("nothing") } // Just so that coverage is triggered.
	j := NewWatchJob(testIfile, ModeSyncthing, runHooks, runHooks, zap.NewNop())

	var (
		walkCount = 0
		mu        sync.Mutex
	)

	j.walk = func() error {
		mu.Lock()
		walkCount++
		mu.Unlock()
		return fmt.Errorf("test walk error")
	}

	err := j.Run()
	require.Error(t, err)

	require.Equal(t, j.Status(), WatchJobStatusFailed)

	require.Equal(t, j.Ifile(), j.ifile)       // Just so that coverage is triggered.
	require.Equal(t, j.ScanPath(), j.scanPath) // Just so that coverage is triggered.
}
