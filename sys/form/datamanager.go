package form

import (
	"io"
	"os"
	"sync"

	"git.proxeus.com/core/central/sys/file"
)

type DataManager struct {
	sync.RWMutex
	DataCluster   map[string]file.MapIO
	BaseFilePath  string
	madeFileInfos bool
}

func NewDataManager(baseFilePath string) *DataManager {
	return &DataManager{
		BaseFilePath: baseFilePath,
		DataCluster:  make(map[string]file.MapIO),
	}
}

func (me *DataManager) OnLoad() {
	me.mkFileInfos()
}

func (me *DataManager) GetDataFile(formID, name string) (*file.IO, error) {
	if formID != "" && name != "" {
		me.RLock()
		if mapIO, ok := me.DataCluster[formID]; ok {
			fi := mapIO.GetFileInfo(me.BaseFilePath, name)
			if fi != nil {
				me.RUnlock()
				return fi, nil
			}
		}
		me.RUnlock()
		return nil, os.ErrNotExist
	}
	return nil, os.ErrInvalid
}

func (me *DataManager) GetDataFiles(formID, baseUri string) (files map[string]*file.IO, err error) {
	if formID != "" {
		me.RLock()
		if formDataMap, ok := me.DataCluster[formID]; ok {
			files = make(map[string]*file.IO)
			for name, maybeFile := range formDataMap {
				if maybeFile != nil {
					if fileInfo, ok := maybeFile.(*file.IO); ok {
						files[name] = fileInfo
					}
				}
			}
		}
		me.RUnlock()
		return
	}
	err = os.ErrInvalid
	return
}

func (me *DataManager) GetDataFileWriter(formID, name string, writer io.Writer) (n int64, f *file.IO, err error) {
	if formID != "" && name != "" {
		f, err = me.GetDataFile(formID, name)
		if err == nil {
			if writer != nil {
				var tmplFile *os.File
				tmplFile, err = os.OpenFile(f.Path(), os.O_RDONLY, 0600)
				if err != nil {
					return
				}
				defer tmplFile.Close()
				var fstat os.FileInfo
				fstat, err = tmplFile.Stat()
				if err != nil {
					return
				}
				n, err = io.CopyN(writer, tmplFile, fstat.Size())
				return
			}
		}
		return
	}
	err = os.ErrInvalid
	return

}

func (me *DataManager) GetData(formID string) (map[string]interface{}, error) {
	if formID != "" {
		me.RLock()
		defer me.RUnlock()
		return me.DataCluster[formID], nil
	}
	return nil, os.ErrInvalid
}

func (me *DataManager) Clear(formID string) error {
	if formID != "" {
		me.RLock()
		defer me.RUnlock()
		delete(me.DataCluster, formID)
		return nil
	}
	return os.ErrInvalid
}

func (me *DataManager) GetDataByPath(formID, dataPath string) (interface{}, error) {
	if formID != "" {
		me.RLock()
		defer me.RUnlock()
		if mapIo, ok := me.DataCluster[formID]; ok {
			return mapIo.Get(dataPath), nil
		}
		return nil, nil
	}
	return nil, os.ErrInvalid
}

/**
returns:
{
	"key":"value",
	"fileKey": {
		"name":"myfile.pdf",
		"contentType":"application/pdf",
		"size":123,
		"path":"{{BaseFilePath}}/76ed295d-92d8-41b1-83be-7078ea9a94a2"
	}
}
*/
func (me *DataManager) GetAllData() (dat map[string]interface{}, err error) {
	me.RLock()
	defer me.RUnlock()
	if len(me.DataCluster) > 0 {
		dat = make(map[string]interface{})
		for _, formDataMap := range me.DataCluster {
			for name, maybeFile := range formDataMap {
				if maybeFile != nil {
					if fileInfo, ok := maybeFile.(*file.IO); ok {
						dat[name] = fileInfo
						continue
					} else if fileMap, ok := maybeFile.(map[string]interface{}); ok {
						if file.IsFileInfo(fileMap) {
							dat[name] = file.FromMap(me.BaseFilePath, fileMap)
							continue
						}
					}
				}
				dat[name] = maybeFile
			}
		}
	}
	return
}

/**
returns:
dat = {
	"key":"value",
	"fileKey": {
		"name":"myfile.pdf",
		"contentType":"application/pdf",
		"size":123,
		"path":"76ed295d-92d8-41b1-83be-7078ea9a94a2"
	}
}
files = [
	"{{BaseFilePath}}/76ed295d-92d8-41b1-83be-7078ea9a94a2",
	...
]
*/
func (me *DataManager) GetAllDataFilePathNameOnly() (dat map[string]interface{}, files []string) {
	me.RLock()
	defer me.RUnlock()
	if len(me.DataCluster) > 0 {
		dat = make(map[string]interface{})
		files = make([]string, 0)
		for _, formDataMap := range me.DataCluster {
			for name, maybeFile := range formDataMap {
				if maybeFile != nil {
					if fileInfo, ok := maybeFile.(*file.IO); ok {
						files = append(files, fileInfo.Path())
						dat[name] = fileInfo.ToMap(true)
						continue
					}
				}
				dat[name] = maybeFile
			}
		}
	}
	return
}

func (me *DataManager) GetAllDataFile(baseUri string) (dat map[string]interface{}, files map[string]*file.IO, err error) {
	me.RLock()
	defer me.RUnlock()
	if len(me.DataCluster) == 0 {
		return
	}
	dat = make(map[string]interface{})
	files = make(map[string]*file.IO)
	for _, formDataMap := range me.DataCluster {
		for name, maybeFile := range formDataMap {
			if maybeFile != nil {
				if fileInfo, ok := maybeFile.(*file.IO); ok {
					files[name] = fileInfo
					continue
				}
			}
			dat[name] = maybeFile
		}
	}
	return
}

func (me *DataManager) PutData(formID string, dat map[string]interface{}) error {
	if formID != "" {
		me.Lock()
		if mapIO, ok := me.DataCluster[formID]; ok {
			mapIO.MergeWith(dat)
		} else {
			mapIO := file.MapIO{}
			mapIO.MergeWith(dat)
			me.DataCluster[formID] = mapIO
		}
		me.Unlock()
		return nil
	}
	return os.ErrInvalid
}

func (me *DataManager) PutDataWithoutMerge(formID string, dat map[string]interface{}) error {
	if formID != "" {
		me.Lock()
		me.DataCluster[formID] = dat
		me.Unlock()
		return nil
	}
	return os.ErrInvalid
}

func (me *DataManager) PutDataFile(formID, name string, f file.Meta, reader io.Reader) (written int64, err error) {
	var existingFileInfo *file.IO
	existingFileInfo, err = me.GetDataFile(formID, name)
	me.Lock()

	var formDataMap map[string]interface{}
	formDataMap = me.DataCluster[formID]
	if formDataMap == nil {
		formDataMap = make(map[string]interface{})
		me.DataCluster[formID] = formDataMap
	}
	if os.IsNotExist(err) || existingFileInfo == nil {
		existingFileInfo = file.New(me.BaseFilePath, f)
		written, err = existingFileInfo.Write(reader)
		formDataMap[name] = existingFileInfo
		me.Unlock()
	} else {
		me.Unlock()
		existingFileInfo.Update(f.Name, f.ContentType)
		written, err = existingFileInfo.Write(reader)
	}
	return
}

func (me *DataManager) mkFileInfos() {
	if !me.madeFileInfos {
		for _, mapIO := range me.DataCluster {
			mapIO.MakeFileInfos(me.BaseFilePath)
		}
		me.madeFileInfos = true
	}
}

func (me *DataManager) Close() (err error) {
	return nil
}
