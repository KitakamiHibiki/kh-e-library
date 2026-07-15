package storage

import (
	"fmt"
	"io"
)

type BaiduDriver struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

func NewBaiduDriver(clientID, clientSecret, refreshToken string) *BaiduDriver {
	return &BaiduDriver{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: refreshToken,
	}
}

func (d *BaiduDriver) Name() string {
	return "baidu"
}

func (d *BaiduDriver) Save(filename string, reader io.Reader) (string, int64, error) {
	return "", 0, fmt.Errorf("baidu netdisk driver: not implemented yet")
}

func (d *BaiduDriver) Open(path string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("baidu netdisk driver: not implemented yet")
}

func (d *BaiduDriver) Delete(path string) error {
	return fmt.Errorf("baidu netdisk driver: not implemented yet")
}

func (d *BaiduDriver) Exists(path string) (bool, error) {
	return false, fmt.Errorf("baidu netdisk driver: not implemented yet")
}

func (d *BaiduDriver) GetURL(path string) (string, bool, error) {
	return "", false, fmt.Errorf("baidu netdisk driver: not implemented yet")
}

func (d *BaiduDriver) Stat(path string) (*FileInfo, error) {
	return nil, fmt.Errorf("baidu netdisk driver: not implemented yet")
}
