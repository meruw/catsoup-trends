package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/meruw/catsoup-trends/internal/research"
	"github.com/meruw/catsoup-trends/internal/server"
	"github.com/meruw/catsoup-trends/web"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"
)

func main() {
	noBrowser := flag.Bool("no-browser", false, "Print the local launch URL without opening a browser")
	data := flag.String("data-dir", "", "Override the local data directory")
	flag.Parse()
	if *data == "" {
		root, err := os.UserConfigDir()
		if err != nil {
			log.Fatal("Cannot locate the user configuration directory")
		}
		*data = filepath.Join(root, "CATsoup Trends")
	}
	if err := os.MkdirAll(*data, 0700); err != nil {
		log.Fatal("Cannot create the Trends data directory")
	}
	store, err := research.Open(filepath.Join(*data, "trends.db"))
	if err != nil {
		log.Fatal("Cannot open the Trends database")
	}
	defer store.Close()
	app, err := server.New(store, *data)
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		log.Fatal("Cannot start the local listener")
	}
	app.Host = listener.Addr().String()
	app.Origin = "http://" + app.Host
	assets, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		log.Fatal("Frontend assets are missing; build the frontend first")
	}
	srv := &http.Server{Handler: app.Handler(assets), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	stopped := make(chan struct{})
	app.Shutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
	go func() {
		defer close(stopped)
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Print("Local server stopped unexpectedly")
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)
	launch := app.Origin + "/#token=" + app.Token
	fmt.Println("CATsoup Trends is running locally. Keep this process open.")
	fmt.Println("Open: " + launch)
	if !*noBrowser {
		if err = openBrowser(launch); err != nil {
			fmt.Println("Open the launch URL above in your browser.")
		}
	}
	cleanup := time.NewTicker(time.Hour)
	defer cleanup.Stop()
	for {
		select {
		case <-stopped:
			return
		case <-signals:
			app.Shutdown()
			<-stopped
			return
		case <-cleanup.C:
			if store.Expire(context.Background(), time.Now()) != nil {
				log.Print("Could not expire cached metadata")
			}
		}
	}
}
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
