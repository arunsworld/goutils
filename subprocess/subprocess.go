package subprocess

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os/exec"
	"time"
)

// SubProcess creates a SubProcess that can be executed.
// Example:
// 	output := make(chan string)
// 	s := subprocess.SubProcess{
// 		Executable: "python",
// 		Arguments:  []string{"main.py"},
// 		Output:     output,
// 		Timeout:    time.Second * 30,
// 	}
// 	if err := s.Run(); err != nil {
// 		log.Fatal(err)
// 	}
// 	c := make(chan os.Signal, 1)
// 	signal.Notify(c, os.Interrupt)
// 	go func() {
// 		sig := <-c
// 		fmt.Println("Got signal:", sig)
// 		s.Kill()
// 	}()
// 	for resp := range output {
// 		fmt.Println(resp)
// 	}
type SubProcess struct {
	Executable string
	Arguments  []string
	Output     chan []byte
	Timeout    time.Duration
	cmd        *exec.Cmd
}

// Run executes the subprocess
func (s *SubProcess) Run() error {
	cmd := exec.Command(s.Executable, s.Arguments...)
	s.cmd = cmd
	cmdout, err := cmd.StdoutPipe()
	if err != nil {
		close(s.Output)
		return err
	}
	cmderr, err := cmd.StderrPipe()
	if err != nil {
		close(s.Output)
		return err
	}
	if err := cmd.Start(); err != nil {
		close(s.Output)
		return errors.New("could not start process: " + err.Error())
	}
	timeout := s.Timeout
	if s.Timeout == 0 {
		timeout = time.Hour
	}

	done := make(chan struct{})
	go scanForContent(cmdout, s.Output, done)
	go scanForContent(cmderr, s.Output, done)

	endTimer := make(chan struct{})
	go func() {
		<-done // wait for cmdout
		<-done // wait for cmderr
		close(s.Output)
		close(endTimer)
		cmd.Wait()
		// log.Println("Job completed...")
	}()

	go func() {
		select {
		case <-endTimer:
		case <-time.After(timeout):
			log.Printf("SubProcess timeout. Killing: %s (%d).\n", s.Executable, cmd.Process.Pid)
			cmd.Process.Kill()
		}
	}()

	return nil
}

// Kill the process
func (s *SubProcess) Kill() {
	log.Printf("Killing due to user input: %s (%d).\n", s.Executable, s.cmd.Process.Pid)
	s.cmd.Process.Kill()
}

// Use NewScanner to read lines as bytes. To avoid a lot of allocation & GC we use 2 byte arrays.
// This is because we're assuming that the output channel will be used by just one goroutine.
// And while one byte slice is being processed the other could be getting copied and then alternated.
// If we directly feed br to output we have a subtle bug. Since byte slice is a pointer it's will start
// getting re-written as soon as the pointer is copied to the channel. This causes corruption at the reading end.
// By copying the bytes we avoid this but we also don't want to allocate new arrays in each loop so we
// toggle between 2 byte arrays.
func scanForContent(r io.Reader, output chan []byte, done chan struct{}) {
	br := bufio.NewScanner(r)
	toggle := true
	var b1 []byte
	var b2 []byte
	for br.Scan() {
		x := br.Bytes()
		if toggle {
			if cap(b1) < len(x) {
				b1 = make([]byte, len(x))
			} else {
				b1 = b1[:len(x)]
			}
			copy(b1, x)
			output <- b1
		} else {
			if cap(b2) < len(x) {
				b2 = make([]byte, len(x))
			} else {
				b2 = b2[:len(x)]
			}
			copy(b2, x)
			output <- b2
		}
		toggle = !toggle
	}
	done <- struct{}{}
}
