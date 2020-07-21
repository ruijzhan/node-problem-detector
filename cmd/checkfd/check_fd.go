package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	OK      = 0
	NONOK   = 1
	UNKNOWN = 2
)

var (
	procPath string
)

func init() {
	flag.StringVar(&procPath, "p", "/proc", "actual path of /proc")
}

func main() {
	flag.Parse()

	maxPath := procPath + "/sys/fs/file-max"
	f, err := os.Open(maxPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(UNKNOWN)
	}
	defer f.Close()
	maxBytes, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(UNKNOWN)
	}
	fdMax, err := strconv.Atoi(strings.TrimSpace(string(maxBytes)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(UNKNOWN)
	}

	files, err := ioutil.ReadDir(procPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(UNKNOWN)
	}

	ch := make(chan int)
	wg := sync.WaitGroup{}
	re := regexp.MustCompile(`[0-9][0-9]*`)
	for _, f := range files {
		if f.IsDir() {
			if re.MatchString(f.Name()) {
				wg.Add(1)
				go countFD(procPath+"/"+f.Name()+"/fd", &wg, ch)
			}
		}
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	fdTotal := 0
	for {
		c, ok := <-ch
		if !ok {
			break
		}
		fdTotal += c
	}

	if fdTotal > fdMax/100*80 {
		fmt.Fprintf(os.Stdout, "current fd usage is %d and max is %d\n", fdTotal, fdMax)
		os.Exit(NONOK)
	} else {
		fmt.Fprintf(os.Stdout, "node has no fd pressure\n")
		os.Exit(OK)
	}
}

func countFD(path string, wg *sync.WaitGroup, fNum chan<- int) {
	defer wg.Done()
	files, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		return
	}
	fNum <- len(files)
}
