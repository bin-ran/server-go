package main

import (
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"server-go/managers"
	"server-go/models"
	"server-go/routers"
	"server-go/utils"
	"strconv"
	"sync"
)

const Version = "0.0.1"

func main() {
	managers.Environment()

	utils.Init(managers.Config.Environment == "development", managers.Config.WebURL)

	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU - 1)

	opts := slog.HandlerOptions{AddSource: true}

	if managers.Config.Environment == "production" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &opts)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &opts)))
	}

	slog.Info("booting", "version", Version, "cores", numCPU)

	wg := sync.WaitGroup{}

	wg.Add(3)

	go managers.InitDB(&wg)
	go managers.InitRedis(&wg)
	go managers.InitRustFSClient(&wg)

	wg.Wait()

	// 初始化基础数据（权限、角色等）
	models.SeedDatabase()

	routers.Init()

	slog.Info("Service Started")

	slog.Info("Listened", "port", managers.Config.Port)

	if err := http.ListenAndServe(":"+strconv.Itoa(managers.Config.Port), nil); err != nil {
		panic(err)
	}
}
