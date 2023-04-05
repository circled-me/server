package storage

import (
	"os"
	"sync"
)

type DiskStorage struct {
	Storage
	// BasePath is a directory (usually mount point of a disk) that is writable by the current process
	BasePath  string
	dirs      map[string]bool // local cache of created dirs
	dirsMutex sync.Mutex
}

func (s *DiskStorage) EnsureDirExists(dir string) error {
	s.dirsMutex.Lock()
	defer s.dirsMutex.Unlock()

	if ok := s.dirs[dir]; ok {
		return nil
	}
	s.dirs[dir] = true
	return os.MkdirAll(dir, 0777)
}

func (s *DiskStorage) GetFullPath(path string) string {
	return s.BasePath + "/" + path
}

func NewDiskStorage(bucket *Bucket) StorageAPI {
	result := &DiskStorage{
		BasePath: bucket.Path,
		Storage: Storage{
			Bucket: *bucket,
		},
		dirs: make(map[string]bool, 10),
	}
	result.specifics = result
	return result
}

func (s *DiskStorage) EnsureLocalFile(path string) error {
	return nil
}

func (s *DiskStorage) ReleaseLocalFile(path string) {
	// noop
}

func (s *DiskStorage) UpdateFile(path, mimeType string) error {
	return nil
}

func (s *DiskStorage) DeleteRemoteFile(path string) {
	// noop
}
