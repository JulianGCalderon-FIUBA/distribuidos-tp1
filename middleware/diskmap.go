package middleware

import (
	"distribuidos/tp1/database"
	"encoding/binary"
	"io"
	"os"
	"path"
	"strconv"
)

const GAMES_DIR string = "games"

type DiskMap struct {
	name string
}

func NewDiskMap(name string) *DiskMap {
	return &DiskMap{
		name: name,
	}
}

func (m *DiskMap) Get(db *database.Database, k string) (*GameStat, error) {
	fileName := m.GamesPath(k)

	file, err := db.Get(fileName)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(file)
	if err != nil {
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

func (m *DiskMap) GetAll(db *database.Database) ([]*GameStat, error) {
	entries, err := db.GetAll(GAMES_DIR)
	if err != nil {
		return nil, err
	}

	stats := make([]*GameStat, 0)

	for _, e := range entries {
		g, err := m.Get(db, path.Base(e))
		if err != nil {
			return nil, err
		}
		stats = append(stats, g)
	}
	return stats, nil
}

func (m *DiskMap) Insert(snapshot *database.Snapshot, stat GameStat) error {

	path := m.GamesPath(strconv.Itoa(int(stat.AppID)))
	file, err := snapshot.Create(path)
	if err != nil {
		return err
	}
	header := struct {
		AppId uint64
		Stats uint64
	}{
		AppId: stat.AppID,
		Stats: stat.Stat,
	}
	err = binary.Write(file, binary.LittleEndian, header)
	if err != nil {
		return nil
	}
	return binary.Write(file, binary.LittleEndian, []byte(stat.Name))
}

func (m *DiskMap) Increment(snapshot *database.Snapshot, id uint64, value uint64) error {

	path := m.GamesPath(strconv.Itoa(int(id)))
	exists, err := snapshot.Exists(path)
	if err != nil {
		return err
	}
	file, err := snapshot.Update(path)
	if err != nil {
		return err
	}

	header := struct {
		AppID uint64
		Stat  uint64
	}{AppID: id, Stat: value}

	if exists {
		err = binary.Read(file, binary.LittleEndian, &header)
		if err != nil {
			return err
		}
		value += header.Stat
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	err = binary.Write(file, binary.LittleEndian, id)
	if err != nil {
		return err
	}
	return binary.Write(file, binary.LittleEndian, value)
}

func (m *DiskMap) Rename(snapshot *database.Snapshot, id uint64, name string) error {

	path := m.GamesPath(strconv.Itoa(int(id)))
	file, err := snapshot.Update(path)
	if os.IsNotExist(err) {
		return m.Insert(snapshot, GameStat{
			AppID: id,
			Name:  name,
		})
	}
	if err != nil {
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
	return binary.Write(file, binary.LittleEndian, []byte(name))
}

// Returns path to specific game inside games folder in database
func (m *DiskMap) GamesPath(k string) string {
	return path.Join(GAMES_DIR, k)
}

/*
func (m *DiskMap) Remove() error {
	return os.RemoveAll(m.name)
}
*/
