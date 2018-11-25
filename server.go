package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"
)

type Option struct {
	Type  string
	Value string
}

type Config map[string]*Option

func (c Config) String() string {
	o := ""
	o += fmt.Sprintf("{\n")
	for i, v := range c {
		o += fmt.Sprintf("	%s (%s):  %v\n", i, v.Type, v.Value)
	}
	o += fmt.Sprintf("}")
	return o
}

func (o *Option) String() string {
	return o.Value
}
func (o *Option) Bool() bool {
	b, err := strconv.ParseBool(o.Value)
	if err != nil {
		panic(err)
	}
	return b
}
func (o *Option) Int() int {
	b, err := strconv.Atoi(o.Value)
	if err != nil {
		panic(err)
	}
	return b
}

func (o *Option) Duration() time.Duration {
	b, err := time.ParseDuration(o.Value)
	if err != nil {
		panic(err)
	}
	return b
}

func (o *Option) SetString(x string) {
	o.Value = x
}

func (o *Option) SetInt(x int) {
	a := strconv.FormatInt(int64(x), 10)
	o.Value = a
}

func (o *Option) SetBool(x bool) {

	o.Value = strconv.FormatBool(x)
}

func (c *Config) Get(x string) *Option {
	if v, ok := (*c)[x]; ok {
		return v
	}
	panic(fmt.Sprintf("couldn't find %s", x))
	/*
		for i := range *c {
			if (*c)[i].Name == x {
				return (*c)[i]
			}
		}
	*/
	return nil
}

func (c *Config) Set(x string, value string) {
	if _, ok := (*c)[x]; !ok {
		panic(fmt.Sprintf("Config.Set: %s doesn't exist in array", x))
	}
	(*c)[x].Value = value
	/*
		for i := range *c {
			if (*c)[i].Name == x {
				(*c)[i].Value = value
			}
		}
	*/
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
		iPrintf("server shutting down in %v\n", MainConfig.Get("ShutdownTimeout").Duration().String())
		ctx, _ := context.WithTimeout(context.Background(), MainConfig.Get("ShutdownTimeout").Duration())
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
