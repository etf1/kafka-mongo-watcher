package signal

import (
	"os"
	"os/signal"
)

// This is a really simple signal subscriber
// Signal subscriber that allows you to attach a callback to an `os.Signal` notification.
// Usefull to react to any os.Signal.
// It returns an `unsubscribe` function that stops the goroutine and clean allocated object
//
// Example:
// unsubscriber := signal.Subscribe(func(s os.Signal) {
//   fmt.Println("process as been asked to be stopped")
// }, os.SIGSTOP)
//
// call "unsubscriber()" in order to detach your callback and clean memory
//
// if no 2nd arg is passed, the `callback` func will be called everytime an os.Signal is triggered
// signal.subscribe(func(s os.Signal) {fmt.Println("called for any signal")})
//
// /!\ CAUTION /!\
// If you call it with second arg to `nil`, no signal will be listened
// signal.subscribe(func(s os.Signal) {fmt.Println("NEVER BE CALLED")}, nil)
func Subscribe(callback func(os.Signal), signals ...os.Signal) func() {
	signalChan := make(chan os.Signal, 1)
	stopChan := make(chan struct{}, 1)
	signal.Notify(signalChan, signals...)
	go func() {
		for {
			select {
			case <-stopChan:
				signal.Stop(signalChan)
				close(stopChan)
				close(signalChan)
				return
			case sig := <-signalChan:
				go callback(sig)
			}
		}
	}()
	return func() {
		stopChan <- struct{}{}
	}
}
