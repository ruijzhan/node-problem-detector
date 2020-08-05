package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
)

const (
	OK      = 0
	NONOK   = 1
	UNKNOWN = 2
)

var (
	procPath           string
	threshold          int
	zombiesTotal       int
	procTotal          int
)

func init()  {
	flag.StringVar(&procPath, "p", "/proc", "actual path of /proc")
	flag.IntVar(&threshold,"t", 20, "warning threshold percentage")
}
func main() {
	flag.Parse()

	files, err := ioutil.ReadDir(procPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(UNKNOWN)
	}

	ch := make(chan int)
	wg := sync.WaitGroup{}
	re := regexp.MustCompile(`[0-9][0-9]*`)

	for _, f := range files {
		if f.IsDir() && re.MatchString(f.Name()) {
			procTotal ++
			wg.Add(1)
			go func(fPath string, wg *sync.WaitGroup, n chan<- int) {
				defer wg.Done()
				if s, err := ioutil.ReadFile(fPath); err != nil {
					fmt.Fprintf(os.Stderr, "%v", err)
				} else {
					if strings.Index(string(s), "State:\tZ") != -1 {
						ch <- 1
					}
				}
			}(procPath+"/"+f.Name()+"/status", &wg, ch)
		}
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for {
		c, ok := <-ch
		if !ok {
			break
		}
		zombiesTotal += c
	}

	fmt.Fprintf(os.Stdout, "zombie processes: %d, total: %d", zombiesTotal, procTotal)

	if procTotal / 100 * threshold < zombiesTotal {
		os.Exit(NONOK)
	} else {
		os.Exit(OK)
	}

}
