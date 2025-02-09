/**
@copyright: fantasysky 2016
@brief: 实现路径相关功能
@author: fanky
@version: 1.0
@date: 2019-01-06
**/

package fspath

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"fsky.pro/fsdef"
)

// 整理路径
func CleanPath(path string) string {
	path = filepath.Clean(path)
	path = strings.ReplaceAll(path, "/", string(os.PathSeparator))
	return strings.ReplaceAll(path, "\\", string(os.PathSeparator))
}

// 获取文件路径的去掉扩展名部分
func FileNoExt(file string) string {
	ext := path.Ext(file)
	return file[:len(file)-len(ext)]
}

// IsPathExists 判断路径是否存在(包括文件和文件夹)
// 注意：
//	如果参数 path 为空字符串，则返回 false
func IsPathExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

// IsDir 判断指定路径，是否是已经存在的路径
// 注意：
//	如果 path 为空字符串，则返回 false，而不会认为是当前路径
func IsDirExists(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// IsFile 判断指定的路径是否是已经存在的文件
func IsFileExists(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

// IsFileAccess 文件是否可被存取
func IsFileAccessable(filePath string) bool {
	file, err := os.OpenFile(filePath, os.O_RDWR, 0666)
	if err != nil { return false }
	fileInfo, err := file.Stat()
	if err != nil { return false }
	return fileInfo.Mode().Perm()&0400 == 0
}

// -------------------------------------------------------------------
// CurrentDir 获取可执行程序当前路径
func ExecuteDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// 判断是否 go run 启动的运行
	tempDir := os.TempDir()
	execDir, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", err
	}
	goRun := strings.HasPrefix(execDir, tempDir)
	if goRun {
		return dir, nil
	}
	return filepath.Abs(filepath.Dir(os.Args[0]))
}

// IsSubFilePathOf
// 判断指定文件或目录是否为另一个目录的子目录或子文件
func IsSubFilePathOf(sub string, parent string) bool {
	relPath, err := filepath.Rel(parent, sub)
	if err != nil { return false }
	return !filepath.IsAbs(relPath) && !strings.HasPrefix(relPath, "..")
}

// -------------------------------------------------------------------
// 遍历指定文件夹下所有子孙文件
// fun 的第一个参数是当前遍历到的文件相对于 root 的路径；第二个参数表示是否是文件夹
// 如果 fun 返回 false 则退出遍历
func WorkDir(root string, fun func(string, fs.FileInfo) bool) error {
	cache := [][2]string{[2]string{root, ""}}
	for len(cache) > 0 {
		subRoot := cache[0]
		cache = cache[1:]
		finfos, err := ioutil.ReadDir(subRoot[0])
		if err != nil { return fmt.Errorf("read dir %q fail", subRoot[0]) }
		for _, finfo := range finfos {
			if finfo.IsDir() {
				root := path.Join(subRoot[0], finfo.Name())
				sub := path.Join(subRoot[1], finfo.Name())
				if !fun(sub, finfo) { return nil }
				cache = append(cache, [2]string{root, sub})
			} else {
				sub := path.Join(subRoot[1], finfo.Name())
				if !fun(sub, finfo) {
					return nil
				}
			}
		}
	}
	return nil
}

// -------------------------------------------------------------------
// CopyFile 复制文件
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return errors.New(fmt.Sprintf("open surce file(%q) fail: %s", src, err.Error()))
	}
	defer srcFile.Close()

	dir, _ := filepath.Split(dst)
	// 如果文件夹部分为空，则认为是当前路径
	if dir == "" {
		dir = "." + string(filepath.Separator)
	} else if !IsDirExists(dir) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return errors.New(fmt.Sprintf("create dst file direcotries(%q) fail: %s", dir, err.Error()))
		}
	}

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return errors.New(fmt.Sprintf("create dst file(%q) fail: %s", dst, err.Error()))
	}
	defer dstFile.Close()

	io.Copy(dstFile, srcFile)
	return nil
}

// CopyDir 复制文件夹
// 如果目标文件夹已经存在，并且不为空，则复制失败
//
//	参数 allowDstHasFile 为 false，表示如果目标文件夹有文件，则复制失败
//	参数 allowDstFasFile 为 true，则允许目标文件夹已经有其他文件，并保留原有文件（但是有重名文件，则复制失败）
func CopyDir(src, dst string, allowDstHasFile bool, out *os.File) error {
	src = filepath.Clean(src)
	// 不允许目标文件夹有文件
	if !allowDstHasFile {
		fs, err := ioutil.ReadDir(dst)
		if err == nil && len(fs) > 0 {
			return errors.New(fmt.Sprintf("dst folder(%q) has been exist and it is not empty", dst))
		}
	}

	// 创建目标文件夹
	if !IsDirExists(dst) {
		err := os.MkdirAll(dst, os.ModePerm)
		if err != nil {
			return errors.New(fmt.Sprintf("create dst folder(%q) fail: %s", dst, err.Error()))
		}
	}

	// 遍历源文件夹
	srcPathLen := len(src)
	err := filepath.Walk(src, func(srcFile string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		dstFile := filepath.Join(dst, srcFile[srcPathLen:])
		if out != nil {
			out.Write([]byte(fmt.Sprintf("%s->%s%s", srcFile, dstFile, fsdef.Endline)))
		}
		return CopyFile(srcFile, dstFile)
	})
	if err != nil {
		return errors.New(fmt.Sprintf("can't read source direcotry(%q): %s", src, err.Error()))
	}
	return nil
}
