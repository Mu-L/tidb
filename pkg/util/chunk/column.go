// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chunk

import (
	"fmt"
	"math"
	"math/bits"
	"math/rand"
	"time"
	"unsafe"

	"github.com/pingcap/tidb/pkg/types"
	"github.com/pingcap/tidb/pkg/util/hack"
)

// For varLenColumn (e.g. varchar), the accurate length of an element is unknown.
// Therefore, in the first executor.Next we use an experience value -- 8 (so it may make runtime.growslice)
const estimatedElemLen = 8

// AppendDuration appends a duration value into this Column.
// Fsp is ignored.
func (c *Column) AppendDuration(dur types.Duration) {
	c.AppendInt64(int64(dur.Duration))
}

// AppendMyDecimal appends a MyDecimal value into this Column.
func (c *Column) AppendMyDecimal(dec *types.MyDecimal) {
	*(*types.MyDecimal)(unsafe.Pointer(&c.elemBuf[0])) = *dec
	c.finishAppendFixed()
}

func (c *Column) appendNameValue(name string, val uint64) {
	var buf [8]byte
	copy(buf[:], (*[8]byte)(unsafe.Pointer(&val))[:])
	c.data = append(c.data, buf[:]...)
	c.data = append(c.data, name...)
	c.finishAppendVar()
}

// AppendJSON appends a BinaryJSON value into this Column.
func (c *Column) AppendJSON(j types.BinaryJSON) {
	c.data = append(c.data, j.TypeCode)
	c.data = append(c.data, j.Value...)
	c.finishAppendVar()
}

// AppendVectorFloat32 appends a VectorFloat32 value into this Column.
func (c *Column) AppendVectorFloat32(v types.VectorFloat32) {
	c.data = v.SerializeTo(c.data)
	c.finishAppendVar()
}

// AppendSet appends a Set value into this Column.
func (c *Column) AppendSet(set types.Set) {
	c.appendNameValue(set.Name, set.Value)
}

// Column stores one column of data in Apache Arrow format.
// See https://arrow.apache.org/docs/format/Columnar.html#format-columnar
type Column struct {
	length     int
	nullBitmap []byte  // bit 0 is null, 1 is not null
	offsets    []int64 // used for varLen column. Row i starts from data[offsets[i]]
	data       []byte
	elemBuf    []byte

	avoidReusing bool // avoid reusing the Column by allocator
}

// ColumnAllocator defines an allocator for Column.
type ColumnAllocator interface {
	NewColumn(ft *types.FieldType, count int) *Column
}

// DefaultColumnAllocator is the default implementation of ColumnAllocator.
type DefaultColumnAllocator struct{}

// NewColumn implements the ColumnAllocator interface.
func (DefaultColumnAllocator) NewColumn(ft *types.FieldType, capacity int) *Column {
	return newColumn(getFixedLen(ft), capacity)
}

// NewEmptyColumn creates a new column with nothing.
func NewEmptyColumn(ft *types.FieldType) *Column {
	elemLen := getFixedLen(ft)
	col := Column{}
	if elemLen != VarElemLen {
		col.elemBuf = make([]byte, elemLen)
	} else {
		col.offsets = append(col.offsets, 0)
	}
	return &col
}

// NewColumn creates a new column with the specific type and capacity.
func NewColumn(ft *types.FieldType, capacity int) *Column {
	return newColumn(getFixedLen(ft), capacity)
}

func newColumn(ts, capacity int) *Column {
	var col *Column
	if ts == VarElemLen {
		col = newVarLenColumn(capacity)
	} else {
		col = newFixedLenColumn(ts, capacity)
	}
	return col
}

// newFixedLenColumn creates a fixed length Column with elemLen and initial data capacity.
func newFixedLenColumn(elemLen, capacity int) *Column {
	return &Column{
		elemBuf:    make([]byte, elemLen),
		data:       make([]byte, 0, getDataMemCap(capacity, elemLen)),
		nullBitmap: make([]byte, 0, getNullBitmapCap(capacity)),
	}
}

// newVarLenColumn creates a variable length Column with initial data capacity.
func newVarLenColumn(capacity int) *Column {
	return &Column{
		offsets:    make([]int64, 1, getOffsetsCap(capacity)),
		data:       make([]byte, 0, getDataMemCap(estimatedElemLen, capacity)),
		nullBitmap: make([]byte, 0, getNullBitmapCap(capacity)),
	}
}

func getDataMemCap(capacity int, elemLen int) int64 {
	return int64(elemLen * capacity)
}

func getNullBitmapCap(capacity int) int64 {
	return int64((capacity + 7) >> 3)
}

func getOffsetsCap(capacity int) int64 {
	return int64(capacity + 1)
}

func (c *Column) typeSize() int {
	if len(c.elemBuf) > 0 {
		return len(c.elemBuf)
	}
	return VarElemLen
}

func (c *Column) isFixed() bool {
	return c.elemBuf != nil
}

// Reset resets this Column according to the EvalType.
// Different from reset, Reset will reset the elemBuf.
func (c *Column) Reset(eType types.EvalType) {
	switch eType {
	case types.ETInt:
		c.ResizeInt64(0, false)
	case types.ETReal:
		c.ResizeFloat64(0, false)
	case types.ETDecimal:
		c.ResizeDecimal(0, false)
	case types.ETString:
		c.ReserveString(0)
	case types.ETDatetime, types.ETTimestamp:
		c.ResizeTime(0, false)
	case types.ETDuration:
		c.ResizeGoDuration(0, false)
	case types.ETJson:
		c.ReserveJSON(0)
	case types.ETVectorFloat32:
		c.ReserveVectorFloat32(0)
	default:
		panic(fmt.Sprintf("invalid EvalType %v", eType))
	}
}

// Rows returns the row number in current column
func (c *Column) Rows() int {
	return c.length
}

// reset resets the underlying data of this Column but doesn't modify its data type.
func (c *Column) reset() {
	c.length = 0
	c.nullBitmap = c.nullBitmap[:0]
	if len(c.offsets) > 0 {
		// The first offset is always 0, it makes slicing the data easier, we need to keep it.
		c.offsets = c.offsets[:1]
	} else if !c.isFixed() {
		c.offsets = append(c.offsets, 0)
	}
	c.data = c.data[:0]
}

// IsNull returns if this row is null.
func (c *Column) IsNull(rowIdx int) bool {
	nullByte := c.nullBitmap[rowIdx/8]
	return nullByte&(1<<(uint(rowIdx)&7)) == 0
}

// CopyConstruct copies this Column to dst.
// If dst is nil, it creates a new Column and returns it.
func (c *Column) CopyConstruct(dst *Column) *Column {
	if dst != nil {
		dst.length = c.length
		dst.nullBitmap = append(dst.nullBitmap[:0], c.nullBitmap...)
		dst.offsets = append(dst.offsets[:0], c.offsets...)
		dst.data = append(dst.data[:0], c.data...)
		dst.elemBuf = append(dst.elemBuf[:0], c.elemBuf...)
		return dst
	}
	newCol := &Column{length: c.length}
	newCol.nullBitmap = append(newCol.nullBitmap, c.nullBitmap...)
	newCol.offsets = append(newCol.offsets, c.offsets...)
	newCol.data = append(newCol.data, c.data...)
	newCol.elemBuf = append(newCol.elemBuf, c.elemBuf...)
	return newCol
}

// AppendNullBitmap append a null/notnull value to the column's null map
func (c *Column) AppendNullBitmap(notNull bool) {
	c.appendNullBitmap(notNull)
}

func (c *Column) appendNullBitmap(notNull bool) {
	idx := c.length >> 3
	if idx >= len(c.nullBitmap) {
		c.nullBitmap = append(c.nullBitmap, 0)
	}
	if notNull {
		pos := uint(c.length) & 7
		c.nullBitmap[idx] |= byte(1 << pos)
	}
}

// AppendCellNTimes append the pos-th Cell in source column to target column N times
func (c *Column) AppendCellNTimes(src *Column, pos, times int) {
	notNull := !src.IsNull(pos)
	if times == 1 {
		c.appendNullBitmap(notNull)
	} else {
		c.appendMultiSameNullBitmap(notNull, times)
	}
	if c.isFixed() {
		elemLen := len(src.elemBuf)
		offset := pos * elemLen
		for range times {
			c.data = append(c.data, src.data[offset:offset+elemLen]...)
		}
	} else {
		start, end := src.offsets[pos], src.offsets[pos+1]
		for range times {
			c.data = append(c.data, src.data[start:end]...)
			c.offsets = append(c.offsets, int64(len(c.data)))
		}
	}
	c.length += times
}

// appendMultiSameNullBitmap appends multiple same bit value to `nullBitMap`.
// notNull means not null.
// num means the number of bits that should be appended.
func (c *Column) appendMultiSameNullBitmap(notNull bool, num int) {
	numNewBytes := ((c.length + num + 7) >> 3) - len(c.nullBitmap)
	b := byte(0)
	if notNull {
		b = 0xff
	}
	for range numNewBytes {
		c.nullBitmap = append(c.nullBitmap, b)
	}
	if !notNull {
		return
	}
	// 1. Set all the remaining bits in the last slot of old c.numBitMap to 1.
	numRemainingBits := uint(c.length % 8)
	bitMask := byte(^((1 << numRemainingBits) - 1))
	c.nullBitmap[c.length/8] |= bitMask
	// 2. Set all the redundant bits in the last slot of new c.numBitMap to 0.
	numRedundantBits := uint(len(c.nullBitmap)*8 - c.length - num)
	bitMask = byte(1<<(8-numRedundantBits)) - 1
	c.nullBitmap[len(c.nullBitmap)-1] &= bitMask
}

// AppendNNulls append n nulls to the column
func (c *Column) AppendNNulls(n int) {
	c.appendMultiSameNullBitmap(false, n)
	if c.isFixed() {
		for range n {
			c.data = append(c.data, c.elemBuf...)
		}
	} else {
		currentLength := c.offsets[c.length]
		for range n {
			c.offsets = append(c.offsets, currentLength)
		}
	}
	c.length += n
}

// AppendNull appends a null value into this Column.
func (c *Column) AppendNull() {
	c.appendNullBitmap(false)
	if c.isFixed() {
		c.data = append(c.data, c.elemBuf...)
	} else {
		c.offsets = append(c.offsets, c.offsets[c.length])
	}
	c.length++
}

func (c *Column) finishAppendFixed() {
	c.data = append(c.data, c.elemBuf...)
	c.appendNullBitmap(true)
	c.length++
}

// AppendInt64 appends an int64 value into this Column.
func (c *Column) AppendInt64(i int64) {
	*(*int64)(unsafe.Pointer(&c.elemBuf[0])) = i
	c.finishAppendFixed()
}

// AppendUint64 appends a uint64 value into this Column.
func (c *Column) AppendUint64(u uint64) {
	*(*uint64)(unsafe.Pointer(&c.elemBuf[0])) = u
	c.finishAppendFixed()
}

// AppendFloat32 appends a float32 value into this Column.
func (c *Column) AppendFloat32(f float32) {
	*(*float32)(unsafe.Pointer(&c.elemBuf[0])) = f
	c.finishAppendFixed()
}

// AppendFloat64 appends a float64 value into this Column.
func (c *Column) AppendFloat64(f float64) {
	*(*float64)(unsafe.Pointer(&c.elemBuf[0])) = f
	c.finishAppendFixed()
}

func (c *Column) finishAppendVar() {
	c.appendNullBitmap(true)
	c.offsets = append(c.offsets, int64(len(c.data)))
	c.length++
}

// AppendString appends a string value into this Column.
func (c *Column) AppendString(str string) {
	c.data = append(c.data, str...)
	c.finishAppendVar()
}

// AppendBytes appends a byte slice into this Column.
func (c *Column) AppendBytes(b []byte) {
	c.data = append(c.data, b...)
	c.finishAppendVar()
}

// AppendTime appends a time value into this Column.
func (c *Column) AppendTime(t types.Time) {
	*(*types.Time)(unsafe.Pointer(&c.elemBuf[0])) = t
	c.finishAppendFixed()
}

// AppendEnum appends a Enum value into this Column.
func (c *Column) AppendEnum(enum types.Enum) {
	c.appendNameValue(enum.Name, enum.Value)
}

const (
	sizeInt64      = int(unsafe.Sizeof(int64(0)))
	sizeUint64     = int(unsafe.Sizeof(uint64(0)))
	sizeUint32     = int(unsafe.Sizeof(uint32(0)))
	sizeFloat32    = int(unsafe.Sizeof(float32(0)))
	sizeFloat64    = int(unsafe.Sizeof(float64(0)))
	sizeMyDecimal  = int(unsafe.Sizeof(types.MyDecimal{}))
	sizeGoDuration = int(unsafe.Sizeof(time.Duration(0)))
	sizeTime       = int(unsafe.Sizeof(types.ZeroTime))
)

var (
	emptyBuf = make([]byte, 4*1024)
)

// resize resizes the column so that it contains n elements, only valid for fixed-length types.
func (c *Column) resize(n, typeSize int, isNull bool) {
	sizeData := n * typeSize
	if cap(c.data) >= sizeData {
		c.data = c.data[:sizeData]
	} else {
		c.data = make([]byte, sizeData)
	}
	if !isNull {
		for j := 0; j < sizeData; j += len(emptyBuf) {
			copy(c.data[j:], emptyBuf)
		}
	}

	newNulls := false
	sizeNulls := (n + 7) >> 3
	if cap(c.nullBitmap) >= sizeNulls {
		c.nullBitmap = c.nullBitmap[:sizeNulls]
	} else {
		c.nullBitmap = make([]byte, sizeNulls)
		newNulls = true
	}
	if !isNull || !newNulls {
		var nullVal, lastByte byte
		if !isNull {
			nullVal = 0xFF
		}

		// Fill the null bitmap
		for i := range c.nullBitmap {
			c.nullBitmap[i] = nullVal
		}
		// Revise the last byte if necessary, when it's not divided by 8.
		if x := (n % 8); x != 0 {
			if !isNull {
				lastByte = byte((1 << x) - 1)
				if len(c.nullBitmap) > 0 {
					c.nullBitmap[len(c.nullBitmap)-1] = lastByte
				}
			}
		}
	}

	if cap(c.elemBuf) >= typeSize {
		c.elemBuf = c.elemBuf[:typeSize]
	} else {
		c.elemBuf = make([]byte, typeSize)
	}

	c.length = n
}

// reserve makes the column capacity be at least enough to contain n elements.
// this method is only valid for var-length types and estElemSize is the estimated size of this type.
func (c *Column) reserve(n, estElemSize int) {
	sizeData := n * estElemSize
	if cap(c.data) >= sizeData {
		c.data = c.data[:0]
	} else {
		c.data = make([]byte, 0, sizeData)
	}

	sizeNulls := (n + 7) >> 3
	if cap(c.nullBitmap) >= sizeNulls {
		c.nullBitmap = c.nullBitmap[:0]
	} else {
		c.nullBitmap = make([]byte, 0, sizeNulls)
	}

	sizeOffs := n + 1
	if cap(c.offsets) >= sizeOffs {
		c.offsets = c.offsets[:1]
	} else {
		c.offsets = make([]int64, 1, sizeOffs)
	}

	c.elemBuf = nil
	c.length = 0
}

// SetNull sets the rowIdx to null.
func (c *Column) SetNull(rowIdx int, isNull bool) {
	if isNull {
		c.nullBitmap[rowIdx>>3] &= ^(1 << uint(rowIdx&7))
	} else {
		c.nullBitmap[rowIdx>>3] |= 1 << uint(rowIdx&7)
	}
}

// SetNulls sets rows in [begin, end) to null.
func (c *Column) SetNulls(begin, end int, isNull bool) {
	i := ((begin + 7) >> 3) << 3
	for ; begin < i && begin < end; begin++ {
		c.SetNull(begin, isNull)
	}
	var v uint8
	if !isNull {
		v = (1 << 8) - 1
	}
	for ; begin+8 <= end; begin += 8 {
		c.nullBitmap[begin>>3] = v
	}
	for ; begin < end; begin++ {
		c.SetNull(begin, isNull)
	}
}

// nullCount returns the number of nulls in this Column.
func (c *Column) nullCount() int {
	var cnt, i int
	for ; i+8 <= c.length; i += 8 {
		// 0 is null and 1 is not null
		cnt += 8 - bits.OnesCount8(c.nullBitmap[i>>3])
	}
	for ; i < c.length; i++ {
		if c.IsNull(i) {
			cnt++
		}
	}
	return cnt
}

// ResizeInt64 resizes the column so that it contains n int64 elements.
func (c *Column) ResizeInt64(n int, isNull bool) {
	c.resize(n, sizeInt64, isNull)
}

// ResizeUint64 resizes the column so that it contains n uint64 elements.
func (c *Column) ResizeUint64(n int, isNull bool) {
	c.resize(n, sizeUint64, isNull)
}

// ResizeFloat32 resizes the column so that it contains n float32 elements.
func (c *Column) ResizeFloat32(n int, isNull bool) {
	c.resize(n, sizeFloat32, isNull)
}

// ResizeFloat64 resizes the column so that it contains n float64 elements.
func (c *Column) ResizeFloat64(n int, isNull bool) {
	c.resize(n, sizeFloat64, isNull)
}

// ResizeDecimal resizes the column so that it contains n decimal elements.
func (c *Column) ResizeDecimal(n int, isNull bool) {
	c.resize(n, sizeMyDecimal, isNull)
}

// ResizeGoDuration resizes the column so that it contains n duration elements.
func (c *Column) ResizeGoDuration(n int, isNull bool) {
	c.resize(n, sizeGoDuration, isNull)
}

// ResizeTime resizes the column so that it contains n Time elements.
func (c *Column) ResizeTime(n int, isNull bool) {
	c.resize(n, sizeTime, isNull)
}

// ReserveString changes the column capacity to store n string elements and set the length to zero.
func (c *Column) ReserveString(n int) {
	c.reserve(n, 8)
}

// ReserveBytes changes the column capacity to store n bytes elements and set the length to zero.
func (c *Column) ReserveBytes(n int) {
	c.reserve(n, 8)
}

// ReserveJSON changes the column capacity to store n JSON elements and set the length to zero.
func (c *Column) ReserveJSON(n int) {
	c.reserve(n, 8)
}

// ReserveVectorFloat32 changes the column capacity to store n vectorFloat32 elements and set the length to zero.
func (c *Column) ReserveVectorFloat32(n int) {
	c.reserve(n, 8)
}

// ReserveSet changes the column capacity to store n set elements and set the length to zero.
func (c *Column) ReserveSet(n int) {
	c.reserve(n, 8)
}

// ReserveEnum changes the column capacity to store n enum elements and set the length to zero.
func (c *Column) ReserveEnum(n int) {
	c.reserve(n, 8)
}

// Int64s returns an int64 slice stored in this Column.
func (c *Column) Int64s() []int64 {
	if len(c.data) == 0 {
		return nil
	}
	return unsafe.Slice((*int64)(unsafe.Pointer(&c.data[0])), c.length)
}

// Uint64s returns a uint64 slice stored in this Column.
func (c *Column) Uint64s() []uint64 {
	if len(c.data) == 0 {
		return nil
	}
	return unsafe.Slice((*uint64)(unsafe.Pointer(&c.data[0])), c.length)
}

// Float32s returns a float32 slice stored in this Column.
func (c *Column) Float32s() []float32 {
	if len(c.data) == 0 {
		return nil
	}
	return unsafe.Slice((*float32)(unsafe.Pointer(&c.data[0])), c.length)
}

// Float64s returns a float64 slice stored in this Column.
func (c *Column) Float64s() []float64 {
	if len(c.data) == 0 {
		return nil
	}
	return unsafe.Slice((*float64)(unsafe.Pointer(&c.data[0])), c.length)
}

// GoDurations returns a Golang time.Duration slice stored in this Column.
// Different from the Row.GetDuration method, the argument Fsp is ignored, so the user should handle it outside.
func (c *Column) GoDurations() []time.Duration {
	if len(c.data) == 0 {
		return nil
	}
	return unsafe.Slice((*time.Duration)(unsafe.Pointer(&c.data[0])), c.length)
}

// Decimals returns a MyDecimal slice stored in this Column.
func (c *Column) Decimals() []types.MyDecimal {
	if len(c.data) == 0 {
		return nil
	}
	return unsafe.Slice((*types.MyDecimal)(unsafe.Pointer(&c.data[0])), c.length)
}

// Times returns a Time slice stored in this Column.
func (c *Column) Times() []types.Time {
	if len(c.data) == 0 {
		return nil
	}
	return unsafe.Slice((*types.Time)(unsafe.Pointer(&c.data[0])), c.length)
}

// GetInt64 returns the int64 in the specific row.
func (c *Column) GetInt64(rowID int) int64 {
	return *(*int64)(unsafe.Pointer(&c.data[rowID*8]))
}

// GetUint64 returns the uint64 in the specific row.
func (c *Column) GetUint64(rowID int) uint64 {
	return *(*uint64)(unsafe.Pointer(&c.data[rowID*8]))
}

// GetFloat32 returns the float32 in the specific row.
func (c *Column) GetFloat32(rowID int) float32 {
	return *(*float32)(unsafe.Pointer(&c.data[rowID*4]))
}

// GetFloat64 returns the float64 in the specific row.
func (c *Column) GetFloat64(rowID int) float64 {
	return *(*float64)(unsafe.Pointer(&c.data[rowID*8]))
}

// GetDecimal returns the decimal in the specific row.
func (c *Column) GetDecimal(rowID int) *types.MyDecimal {
	return (*types.MyDecimal)(unsafe.Pointer(&c.data[rowID*types.MyDecimalStructSize]))
}

// GetString returns the string in the specific row.
func (c *Column) GetString(rowID int) string {
	return string(hack.String(c.data[c.offsets[rowID]:c.offsets[rowID+1]]))
}

// GetJSON returns the JSON in the specific row.
func (c *Column) GetJSON(rowID int) types.BinaryJSON {
	start := c.offsets[rowID]
	return types.BinaryJSON{TypeCode: c.data[start], Value: c.data[start+1 : c.offsets[rowID+1]]}
}

// GetVectorFloat32 returns the VectorFloat32 in the specific row.
func (c *Column) GetVectorFloat32(rowID int) types.VectorFloat32 {
	data := c.data[c.offsets[rowID]:c.offsets[rowID+1]]
	v, _, err := types.ZeroCopyDeserializeVectorFloat32(data)
	if err != nil {
		panic(err)
	}
	return v
}

// GetBytes returns the byte slice in the specific row.
func (c *Column) GetBytes(rowID int) []byte {
	return c.data[c.offsets[rowID]:c.offsets[rowID+1]]
}

// GetEnum returns the Enum in the specific row.
func (c *Column) GetEnum(rowID int) types.Enum {
	name, val := c.getNameValue(rowID)
	return types.Enum{Name: name, Value: val}
}

// GetSet returns the Set in the specific row.
func (c *Column) GetSet(rowID int) types.Set {
	name, val := c.getNameValue(rowID)
	return types.Set{Name: name, Value: val}
}

// GetTime returns the Time in the specific row.
func (c *Column) GetTime(rowID int) types.Time {
	return *(*types.Time)(unsafe.Pointer(&c.data[rowID*sizeTime]))
}

// GetDuration returns the Duration in the specific row.
func (c *Column) GetDuration(rowID int, fillFsp int) types.Duration {
	dur := *(*int64)(unsafe.Pointer(&c.data[rowID*8]))
	return types.Duration{Duration: time.Duration(dur), Fsp: fillFsp}
}

func (c *Column) getNameValue(rowID int) (string, uint64) {
	start, end := c.offsets[rowID], c.offsets[rowID+1]
	if start == end {
		return "", 0
	}
	var val uint64
	copy((*[8]byte)(unsafe.Pointer(&val))[:], c.data[start:])
	return string(hack.String(c.data[start+8 : end])), val
}

// GetRaw returns the underlying raw bytes in the specific row.
func (c *Column) GetRaw(rowID int) []byte {
	var data []byte
	if c.isFixed() {
		elemLen := len(c.elemBuf)
		data = c.data[rowID*elemLen : rowID*elemLen+elemLen]
	} else {
		data = c.data[c.offsets[rowID]:c.offsets[rowID+1]]
	}
	return data
}

// SetRaw sets the raw bytes for the rowIdx-th element.
// NOTE: Two conditions must be satisfied before calling this function:
// 1. The column should be stored with variable-length elements.
// 2. The length of the new element should be exactly the same as the old one.
func (c *Column) SetRaw(rowID int, bs []byte) {
	copy(c.data[c.offsets[rowID]:c.offsets[rowID+1]], bs)
}

// reconstruct reconstructs this Column by removing all filtered rows in it according to sel.
func (c *Column) reconstruct(sel []int) {
	if sel == nil {
		return
	}
	if c.isFixed() {
		elemLen := len(c.elemBuf)
		for dst, src := range sel {
			idx := dst >> 3
			pos := uint16(dst & 7)
			if c.IsNull(src) {
				c.nullBitmap[idx] &= ^byte(1 << pos)
			} else {
				copy(c.data[dst*elemLen:dst*elemLen+elemLen], c.data[src*elemLen:src*elemLen+elemLen])
				c.nullBitmap[idx] |= byte(1 << pos)
			}
		}
		c.data = c.data[:len(sel)*elemLen]
	} else {
		tail := 0
		for dst, src := range sel {
			idx := dst >> 3
			pos := uint(dst & 7)
			if c.IsNull(src) {
				c.nullBitmap[idx] &= ^byte(1 << pos)
				c.offsets[dst+1] = int64(tail)
			} else {
				start, end := c.offsets[src], c.offsets[src+1]
				copy(c.data[tail:], c.data[start:end])
				tail += int(end - start)
				c.offsets[dst+1] = int64(tail)
				c.nullBitmap[idx] |= byte(1 << pos)
			}
		}
		c.data = c.data[:tail]
		c.offsets = c.offsets[:len(sel)+1]
	}
	c.length = len(sel)

	// clean nullBitmap
	c.nullBitmap = c.nullBitmap[:(len(sel)+7)>>3]
	idx := len(sel) >> 3
	if idx < len(c.nullBitmap) {
		pos := uint16(len(sel) & 7)
		c.nullBitmap[idx] &= byte((1 << pos) - 1)
	}
}

// CopyReconstruct copies this Column to dst and removes unselected rows.
// If dst is nil, it creates a new Column and returns it.
func (c *Column) CopyReconstruct(sel []int, dst *Column) *Column {
	if sel == nil {
		return c.CopyConstruct(dst)
	}

	selLength := len(sel)
	if selLength == c.length {
		// The variable 'ascend' is used to check if the sel array is in ascending order
		ascend := true
		for i := 1; i < selLength; i++ {
			if sel[i] < sel[i-1] {
				ascend = false
				break
			}
		}
		if ascend {
			return c.CopyConstruct(dst)
		}
	}

	if dst == nil {
		dst = newColumn(c.typeSize(), len(sel))
	} else {
		dst.reset()
	}

	if c.isFixed() {
		elemLen := len(c.elemBuf)
		dst.elemBuf = make([]byte, elemLen)
		for _, i := range sel {
			dst.appendNullBitmap(!c.IsNull(i))
			dst.data = append(dst.data, c.data[i*elemLen:i*elemLen+elemLen]...)
			dst.length++
		}
	} else {
		dst.elemBuf = nil
		if len(dst.offsets) == 0 {
			dst.offsets = append(dst.offsets, 0)
		}
		for _, i := range sel {
			dst.appendNullBitmap(!c.IsNull(i))
			start, end := c.offsets[i], c.offsets[i+1]
			dst.data = append(dst.data, c.data[start:end]...)
			dst.offsets = append(dst.offsets, int64(len(dst.data)))
			dst.length++
		}
	}
	return dst
}

// MergeNulls merges these columns' null bitmaps.
// For a row, if any column of it is null, the result is null.
// It works like: if col1.IsNull || col2.IsNull || col3.IsNull.
// The caller should ensure that all these columns have the same
// length, and data stored in the result column is fixed-length type.
func (c *Column) MergeNulls(cols ...*Column) {
	if !c.isFixed() {
		panic("result column should be fixed-length type")
	}
	for _, col := range cols {
		if c.length != col.length {
			panic(fmt.Sprintf("should ensure all columns have the same length, expect %v, but got %v", c.length, col.length))
		}
	}
	for _, col := range cols {
		for i := range c.nullBitmap {
			// bit 0 is null, 1 is not null, so do AND operations here.
			c.nullBitmap[i] &= col.nullBitmap[i]
		}
	}
}

// DestroyDataForTest destroies data in the column so that
// we can test if data in column are deeply copied.
func (c *Column) DestroyDataForTest() {
	dataByteNum := len(c.data)
	for i := range dataByteNum {
		c.data[i] = byte(rand.Intn(256))
	}
}

// ContainsVeryLargeElement checks if the column contains element whose length is greater than math.MaxUint32.
func (c *Column) ContainsVeryLargeElement() bool {
	if c.length == 0 {
		return false
	}
	if c.isFixed() {
		return false
	}
	if c.offsets[c.length] <= math.MaxUint32 {
		return false
	}
	for i := range c.length {
		if c.offsets[i+1]-c.offsets[i] > math.MaxUint32 {
			return true
		}
	}
	return false
}
