package docker

// File: honey_server/utils/docker/parse_manifest.go
// Description: Docker镜像元数据解析工具，提供镜像文件的ID、名称、标签等信息提取功能

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const manifestFile = "manifest.json" // Docker镜像清单文件名，用于存储镜像元数据

// ParseImageMetadata 解析Docker镜像文件的元数据
func ParseImageMetadata(filePath string) (string, string, string, error) {
	// 打开镜像文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", "", err
	}
	defer file.Close()

	var reader io.Reader = file

	// 处理gzip压缩的镜像文件
	if strings.HasSuffix(filePath, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return "", "", "", err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// 创建tar包读取器解析镜像文件
	tarReader := tar.NewReader(reader)

	var imageID, imageName, imageTag string

	// 遍历tar包中的文件条目
	for {
		header, err := tarReader.Next()
		if err == io.EOF { // 到达tar包末尾
			break
		}
		if err != nil {
			return "", "", "", err
		}

		// 查找并解析manifest.json文件
		switch header.Name {
		case manifestFile:
			// 读取manifest.json文件内容
			manifestData, err := io.ReadAll(tarReader)
			if err != nil {
				return "", "", "", err
			}
			// 解析manifest内容提取镜像元数据
			data, err := extractImage(string(manifestData))
			if err != nil {
				return "", "", "", err
			}
			return data.ImageID, data.ImageName, data.ImageTag, nil
		}
	}

	// 校验元数据完整性
	if imageID == "" || imageName == "" || imageTag == "" {
		return "", "", "", fmt.Errorf("无法从镜像文件中提取完整的元数据")
	}

	return imageID, imageName, imageTag, nil
}

// manifestType Docker镜像manifest.json文件的结构定义
// 对应manifest文件中的单个镜像配置项
type manifestType struct {
	Config   string   `json:"Config"`   // 镜像配置文件路径
	RepoTags []string `json:"RepoTags"` // 镜像仓库标签列表
}

// manifestData 镜像元数据结构体
// 存储解析后的镜像核心信息
type manifestData struct {
	ImageID   string // 镜像ID
	ImageName string // 镜像名称
	ImageTag  string // 镜像标签
}

// extractImage 解析manifest.json内容提取镜像元数据
// manifest: manifest.json文件的字符串内容
func extractImage(manifest string) (data manifestData, err error) {
	// 解析manifest JSON数据
	var t []manifestType
	err = json.Unmarshal([]byte(manifest), &t)
	if err != nil {
		err = fmt.Errorf("解析manifest文件失败 %s", err)
		return
	}
	// 校验manifest数据有效性
	if len(t) == 0 {
		err = fmt.Errorf("解析manifest文件内容失败 %s", manifest)
		return
	}
	// 提取镜像名称和标签（RepoTags格式：name:tag）
	repoTags := t[0].RepoTags[0]
	_list := strings.Split(repoTags, ":")
	data.ImageName = _list[0]
	data.ImageTag = _list[1]
	// 从Config路径提取镜像ID（Config格式：<hash>/config.json）
	data.ImageID = strings.Split(t[0].Config, "/")[2][:12]
	return
}
