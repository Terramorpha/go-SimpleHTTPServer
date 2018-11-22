package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
)

func (c *Config) Get(x string) *Setting {
	for i := range *c {
		if (*c)[i].Name == x {
			return (*c)[i]
		}
	}
	return nil
}

func (c *Config) Set(x string, value string) *Setting {
	for i := range *c {
		if (*c)[i].Name == x {
			(*c)[i].Value = value
		}
	}
	panic(fmt.Sprintf("Config.Set: %s doesn't exist in array", x))
}

func WaitXInterrupt(x int, c chan os.Signal) chan struct{} {
	ret := make(chan struct{})
	go func() {
		for i := x; i >= 0; i-- {
			<-c
			iPrintf("\n%d interrupts remaining before force shutdown\n", i)
		}
		ret <- struct{}{}
	}()
	return ret
}

func waitTillDone(f func() error) chan error {
	o := make(chan error)
	go func() {
		o <- f()
	}()
	return o
}

func ManageServer(server *http.Server) chan int {
	done := make(chan int)
	go func(ret chan int) {
		channel := make(chan os.Signal)
		signal.Notify(channel, os.Interrupt)
		<-channel
		dPrintln("interrupt")
		fmt.Printf("server shutting down in %v\n", shutdowmTimeout.String())
		ctx, _ := context.WithTimeout(context.Background(), shutdowmTimeout)
		select {
		case <-WaitXInterrupt(10, channel):
			iPrintln("server shutdown forcefully")
		case err := <-waitTillDone(func() error { return server.Shutdown(ctx) }):
			if err != nil {
				dPrintln(err)
				ePrintln("server shutdown forcefully")
			} else {
				iPrintln("server shutdown cleanly")

			}
		}
		ret <- 0
	}(done)
	return done
}

type Config []*Option
