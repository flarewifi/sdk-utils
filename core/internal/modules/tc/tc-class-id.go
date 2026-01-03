package tc

import (
	"fmt"
	"sort"

	jobque "core/utils/job-que"
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
	classQue     = jobque.NewJobQue[interface{}]()
)

type TcClassId uint

func (self TcClassId) String() string {
	return fmt.Sprintf("1:%x", int(self))
}

func (self TcClassId) Uint() uint {
	return uint(self)
}

func (self TcClassId) Cancel() {
	classQue.Exec(func() (interface{}, error) {
		tmpClassIds = removeClassId(tmpClassIds, self)
		return nil, nil
	})
}

func (self TcClassId) Commit() {
	classQue.Exec(func() (interface{}, error) {
		tmpClassIds = removeClassId(tmpClassIds, self)
		usedClassIds = append(usedClassIds, self)
		return nil, nil
	})
}

func (self TcClassId) Restore() {
	classQue.Exec(func() (interface{}, error) {
		usedClassIds = removeClassId(usedClassIds, self)
		return nil, nil
	})
}

func GetAvailableId() TcClassId {
	result, _ := classQue.Exec(func() (interface{}, error) {
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
