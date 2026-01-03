package tc

// import (
// "testing"

// "github.com/stretchr/testify/assert"
// )

// func Test_TcClassId_GetAvailableId(t *testing.T) {
// usedClassIds = []TcClassId{}
// assert.Equal(t, startId, GetAvailableId())

// usedClassIds = []TcClassId{startId + 1, 10}
// assert.Equal(t, startId, GetAvailableId())

// usedClassIds = []TcClassId{startId, startId + 1, startId + 3}
// assert.Equal(t, TcClassId(startId+2), GetAvailableId())

// usedClassIds = []TcClassId{startId, startId + 1}
// assert.Equal(t, TcClassId(startId+2), GetAvailableId())
// }

// func Test_TcClassId_InsertUsedId(t *testing.T) {
// usedClassIds = []TcClassId{startId}
// InsertUsedId(5)
// assert.Equal(t, []TcClassId{startId, 5}, usedClassIds)

// InsertUsedId(10)
// assert.Equal(t, []TcClassId{startId, 5, 10}, usedClassIds)

// InsertUsedId(6)
// assert.Equal(t, []TcClassId{startId, 5, 6, 10}, usedClassIds)

// }

// func Test_TcClassId_RemoveUsedId(t *testing.T) {
// usedClassIds = []TcClassId{10, 20, 30, 40, 50}

// ReturnClassId(10)
// assert.Equal(t, []TcClassId{20, 30, 40, 50}, usedClassIds)

// ReturnClassId(50)
// assert.Equal(t, []TcClassId{20, 30, 40}, usedClassIds)

// ReturnClassId(30)
// assert.Equal(t, []TcClassId{20, 40}, usedClassIds)
// }
