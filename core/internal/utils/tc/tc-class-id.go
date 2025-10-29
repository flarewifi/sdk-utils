package tc

import (
	"fmt"
	"sort"
	"sync"

	jobque "tools/job-que"
)

const (
	TcClassIdRoot TcClassId = iota
	TcClassIdDefault
	TcClassIdUser
	startId
)

var (
	usedClassIds []TcClassId
	tmpClassIds  []TcClassId
	classQue     sync.Mutex
)

type TcClassId uint

func (self TcClassId) String() string {
	return fmt.Sprintf("1:%x", int(self))
}

func (self TcClassId) Uint() uint {
	return uint(self)
}

func (self TcClassId) Cancel() {
	jobque.Exec(&classQue, func() (interface{}, error) {
		tmpClassIds = removeClassId(tmpClassIds, self)
		return nil, nil
	})
}

func (self TcClassId) Commit() {
	jobque.Exec(&classQue, func() (interface{}, error) {
		tmpClassIds = removeClassId(tmpClassIds, self)
		usedClassIds = append(usedClassIds, self)
		return nil, nil
	})
}

func (self TcClassId) Restore() {
	jobque.Exec(&classQue, func() (interface{}, error) {
		usedClassIds = removeClassId(usedClassIds, self)
		return nil, nil
	})
}

func GetAvailableId() TcClassId {
	result, _ := jobque.Exec(&classQue, func() (interface{}, error) {
		classids := orderedIds()

		for i := 0; i < len(classids); i++ {
			expected := (i * 2) + int(startId)
			if classids[i] != TcClassId(expected) {
				return TcClassId(expected), nil
			}
		}

		classid := TcClassId((len(classids) * 2) + int(startId))
		tmpClassIds = append(tmpClassIds, classid)
		return classid, nil
	})

	return result.(TcClassId)
}

func removeClassId(classids []TcClassId, id TcClassId) []TcClassId {
	for i, curr := range classids {
		if id == curr {
			classids = append(classids[:i], classids[i+1:]...)
			break
		}
	}
	return classids
}

func orderedIds() []TcClassId {
	classids := []TcClassId{}

	for _, id := range tmpClassIds {
		classids = append(classids, id)
	}

	for _, id := range usedClassIds {
		classids = append(classids, id)
	}

	sort.Slice(classids, func(i, j int) bool {
		return classids[i] < classids[j]
	})

	return classids
}
