package storage

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type DiskStorage struct {
	Storage
	// BasePath is a directory (usually mount point of a disk) that is writable by the current process
	BasePath  string
	dirs      map[string]bool
	dirsMutex sync.Mutex
}

// func (s *DiskStorage) CreateSubDir(dir string) error {
// 	return os.Mkdir(s.BasePath+dir, 0777)
// }

func (s *DiskStorage) createDir(dir string) error {
	s.dirsMutex.Lock()
	defer s.dirsMutex.Unlock()

	if ok := s.dirs[dir]; ok {
		return nil
	}
	s.dirs[dir] = true
	return os.MkdirAll(dir, 0777)
}

func (s *DiskStorage) getFullPath(path string) string {
	return s.BasePath + "/" + path
}

func (s *DiskStorage) Save(path string, reader io.Reader) (int64, error) {
	fileName := s.getFullPath(path)
	if err := s.createDir(filepath.Dir(fileName)); err != nil {
		return 0, err
	}
	file, err := os.Create(fileName)
	if err != nil {
		return 0, err
	}
	result, err := io.Copy(file, reader)
	file.Close()
	return result, err
}

func (s *DiskStorage) Load(path string, writer io.Writer) (int64, error) {
	fileName := s.getFullPath(path)
	file, err := os.Open(fileName)
	if err != nil {
		return 0, err
	}
	result, err := io.Copy(writer, file)
	file.Close()
	return result, err
}

func (s *DiskStorage) Serve(path string, request *http.Request, writer http.ResponseWriter) {
	fileName := s.getFullPath(path)
	http.ServeFile(writer, request, fileName)
}

func (s *DiskStorage) Delete(path string) error {
	return os.Remove(s.getFullPath(path))
}

func NewDiskStorage(bucket *Bucket) StorageAPI {
	return &DiskStorage{
		BasePath: bucket.Path,
		Storage: Storage{
			Bucket: *bucket,
		},
		dirs: make(map[string]bool, 10),
	}
}
