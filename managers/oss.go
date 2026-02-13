package managers

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Scorpio69t/rustfs-go"
	"github.com/Scorpio69t/rustfs-go/bucket"
	"github.com/Scorpio69t/rustfs-go/pkg/credentials"
	"github.com/Scorpio69t/rustfs-go/types"
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

	client, err := rustfs.New(
		Config.OSS.Endpoint,
		&rustfs.Options{
			Credentials:  credentials.NewStaticV4(Config.OSS.AccessKey, Config.OSS.SecretAccessKey, ""),
			Secure:       true,
			BucketLookup: types.BucketLookupPath, // 设置桶的拼接方式为路径domin/bucketName/xxx，默认情况自动，会将bucket作为子域名
		},
	)

	if err != nil {
		panic(err)
	}

	client.Bucket().Create(ctx, Config.OSS.BucketName, bucket.WithRegion("binran-t1"))

	RustFSClient = &RustFS{Client: client, Bucket: Config.OSS.BucketName}

	slog.Info("RustFS client initialized")

	wg.Done()
}

func (rustfs *RustFS) PresignGet(ctx context.Context, bucketName string, objectName string, expires time.Duration) (string, error) {
	url, _, err := rustfs.Client.Object().PresignGet(ctx, bucketName, objectName, expires, nil)
	return url.String(), err
}

func (rustfs *RustFS) PresignPut(ctx context.Context, bucketName string, objectName string, expires time.Duration) (string, error) {
	url, _, err := rustfs.Client.Object().PresignPut(ctx, bucketName, objectName, expires, nil)
	return url.String(), err
}
