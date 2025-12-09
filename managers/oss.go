package managers

import (
	"context"
	"log/slog"
	"sync"

	"github.com/chenleijava/rustfs-client/rustfs"
)

type OSSConfig struct {
	Endpoint        string `toml:"endpoint"`
	AccessKey       string `toml:"accessKey"`
	SecretAccessKey string `toml:"secretAccessKey"`
	Secure          bool   `toml:"secure"`
	BucketName      string `toml:"bucketName"`
}

type RustFS struct {
	Client *rustfs.Client
	Bucket string
}

var RustFSClient *RustFS

func InitRustFSClient(wg *sync.WaitGroup) {
	ctx := context.Background()

	client, err := rustfs.NewRustFSClient(Config.OSS.Endpoint, Config.OSS.AccessKey, Config.OSS.SecretAccessKey)
	if err != nil {
		panic(err)
	}

	if err := client.CreateBucket(ctx, Config.OSS.BucketName, "binran-t1"); err != nil {
		slog.Error("Failed to create bucket", "error", err)
	}

	RustFSClient = &RustFS{Client: client, Bucket: Config.OSS.BucketName}

	slog.Info("RustFS client initialized")

	wg.Done()
}
