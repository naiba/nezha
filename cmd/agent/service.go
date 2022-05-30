//go:build linux || darwin
// +build linux darwin

package main

func installService() error {
	return nil
}
func removeService() error {
	return nil
}
func startService() error {
	return nil
}

func controlServiceStop() error {
	return nil
}
func controlServicePause() error {
	return nil
}
func controlServiceContinue() error {
	return nil
}
func runService(isDebug bool) {
}
func runSvc() {
	run()
}
