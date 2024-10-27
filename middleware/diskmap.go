package middleware

import (
	"encoding/binary"
	"os"
	"path"
	"strconv"
)

type DiskMap struct {
	name string
	m    map[uint64]GameStat
}

func NewDiskMap(name string) (*DiskMap, error) {
	err := os.MkdirAll(name, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &DiskMap{
		name: name,
		m:    make(map[uint64]GameStat),
	}, nil
}

func (m *DiskMap) Get(id uint64) (*GameStat, error) {
	fileName := path.Join(m.name, strconv.Itoa(int(id)))

	content, err := os.ReadFile(fileName)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var header struct {
		AppId uint64
		Stat  uint64
	}
	n, err := binary.Decode(content, binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}

	return &GameStat{
		AppID: header.AppId,
		Stat:  header.Stat,
		Name:  string(content[n:]),
	}, nil
}

func (m *DiskMap) GetAll() ([]GameStat, error) {

	entries, err := os.ReadDir(m.name)
	if err != nil {
		return nil, err
	}

	games := make([]GameStat, 0)

	for _, e := range entries {
		n, err := strconv.Atoi(e.Name())
		if err != nil {
			return nil, err
		}
		g, err := m.Get(uint64(n))
		if err != nil {
			return nil, err
		}
		games = append(games, *g)
	}

	return games, nil
}

func (m *DiskMap) Insert(stat GameStat) error {
	fileName := path.Join(m.name, strconv.Itoa(int(stat.AppID)))

	header := struct {
		AppId uint64
		Stat  uint64
	}{
		AppId: stat.AppID,
		Stat:  stat.Stat,
	}
	content, err := binary.Append(nil, binary.LittleEndian, &header)
	if err != nil {
		return err
	}
	content = append(content, []byte(stat.Name)...)

	return os.WriteFile(fileName, content, 0644)
}

func (m *DiskMap) Increment(id uint64, value uint64) error {
	fileName := path.Join(m.name, strconv.Itoa(int(id)))
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if os.IsNotExist(err) {
		return m.Insert(GameStat{
			AppID: id,
			Name:  "",
			Stat:  value,
		})
	} else if err != nil {
		return err
	}

	offset := int64(binary.Size(id))
	_, err = file.Seek(offset, 0)
	if err != nil {
		return err
	}

	var stat uint64
	err = binary.Read(file, binary.LittleEndian, &stat)
	if err != nil {
		return err
	}

	_, err = file.Seek(offset, 0)
	if err != nil {
		return err
	}
	stat += value
	return binary.Write(file, binary.LittleEndian, &stat)
}

func (m *DiskMap) Rename(id uint64, name string) error {
	fileName := path.Join(m.name, strconv.Itoa(int(id)))
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if os.IsNotExist(err) {
		return m.Insert(GameStat{
			AppID: id,
			Name:  name,
		})
	} else if err != nil {
		return err
	}

	var header struct {
		AppId uint64
		Stat  uint64
	}
	offset := int64(binary.Size(header))
	_, err = file.Seek(offset, 0)
	if err != nil {
		return err
	}
	err = file.Truncate(offset)
	if err != nil {
		return err
	}

	_, err = file.WriteString(name)
	return err
}
