package foundation

import "cmp"

// SetEquals is setEquals: nil stands for Dart's null (so nil and an empty
// set differ).
func SetEquals[T comparable](a, b map[T]struct{}) bool {
	if a == nil {
		return b == nil
	}
	if b == nil || len(a) != len(b) {
		return false
	}
	for v := range a {
		if _, ok := b[v]; !ok {
			return false
		}
	}
	return true
}

// ListEquals is listEquals: element-wise ==; nil stands for Dart's null.
func ListEquals[T comparable](a, b []T) bool {
	if a == nil {
		return b == nil
	}
	if b == nil || len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MapEquals is mapEquals; nil stands for Dart's null.
func MapEquals[T, U comparable](a, b map[T]U) bool {
	if a == nil {
		return b == nil
	}
	if b == nil || len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// BinarySearch is binarySearch: the index of value in sortedList, or -1.
func BinarySearch[T cmp.Ordered](sortedList []T, value T) int {
	lo, hi := 0, len(sortedList)
	for lo < hi {
		mid := lo + ((hi - lo) >> 1)
		c := cmp.Compare(sortedList[mid], value)
		if c == 0 {
			return mid
		}
		if c < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return -1
}

const kMergeSortLimit = 32

// MergeSortOptions are mergeSort's named arguments. End nil means the list's
// length; Compare nil means the elements' natural order (ordered types, or
// types with a CompareTo(T) int method, like Dart's Comparable).
type MergeSortOptions[T any] struct {
	Start   int
	End     *int
	Compare func(a, b T) int
}

// MergeSort is mergeSort: a stable sort of list[start:end], ported as is
// (insertion sort below 32 elements, merge sort above).
func MergeSort[T any](list []T, opts ...MergeSortOptions[T]) {
	var o MergeSortOptions[T]
	if len(opts) > 0 {
		o = opts[0]
	}
	start, end := o.Start, len(list)
	if o.End != nil {
		end = *o.End
	}
	compare := o.Compare
	if compare == nil {
		compare = defaultCompare[T]
	}
	length := end - start
	if length < 2 {
		return
	}
	if length < kMergeSortLimit {
		insertionSort(list, compare, start, end)
		return
	}
	middle := start + ((end - start) >> 1)
	firstLength := middle - start
	secondLength := end - middle
	scratch := make([]T, secondLength)
	for i := range scratch {
		scratch[i] = list[start]
	}
	mergeSortRec(list, compare, middle, end, scratch, 0)
	firstTarget := end - firstLength
	mergeSortRec(list, compare, start, middle, list, firstTarget)
	merge(compare, list, firstTarget, end, scratch, 0, secondLength, list, start)
}

func defaultCompare[T any](a, b T) int {
	switch x := any(a).(type) {
	case int:
		return cmp.Compare(x, any(b).(int))
	case int64:
		return cmp.Compare(x, any(b).(int64))
	case float64:
		return cmp.Compare(x, any(b).(float64))
	case string:
		return cmp.Compare(x, any(b).(string))
	case interface{ CompareTo(T) int }:
		return x.CompareTo(b)
	}
	panic("foundation: mergeSort needs a Compare function for this element type")
}

// setRange is Dart's List.setRange(start, end, from, skip), which copies
// correctly even when from and list overlap.
func setRange[T any](list []T, start, end int, from []T, skip int) {
	copy(list[start:end], from[skip:skip+end-start])
}

func insertionSort[T any](list []T, compare func(T, T) int, start, end int) {
	for pos := start + 1; pos < end; pos++ {
		lo, hi := start, pos
		element := list[pos]
		for lo < hi {
			mid := lo + ((hi - lo) >> 1)
			if compare(element, list[mid]) < 0 {
				hi = mid
			} else {
				lo = mid + 1
			}
		}
		setRange(list, lo+1, pos+1, list, lo)
		list[lo] = element
	}
}

func movingInsertionSort[T any](list []T, compare func(T, T) int, start, end int, target []T, targetOffset int) {
	length := end - start
	if length == 0 {
		return
	}
	target[targetOffset] = list[start]
	for i := 1; i < length; i++ {
		element := list[start+i]
		lo, hi := targetOffset, targetOffset+i
		for lo < hi {
			mid := lo + ((hi - lo) >> 1)
			if compare(element, target[mid]) < 0 {
				hi = mid
			} else {
				lo = mid + 1
			}
		}
		setRange(target, lo+1, targetOffset+i+1, target, lo)
		target[lo] = element
	}
}

func mergeSortRec[T any](list []T, compare func(T, T) int, start, end int, target []T, targetOffset int) {
	length := end - start
	if length < kMergeSortLimit {
		movingInsertionSort(list, compare, start, end, target, targetOffset)
		return
	}
	middle := start + (length >> 1)
	firstLength := middle - start
	secondLength := end - middle
	targetMiddle := targetOffset + firstLength
	mergeSortRec(list, compare, middle, end, target, targetMiddle)
	mergeSortRec(list, compare, start, middle, list, middle)
	merge(compare, list, middle, middle+firstLength, target, targetMiddle, targetMiddle+secondLength, target, targetOffset)
}

func merge[T any](compare func(T, T) int, firstList []T, firstStart, firstEnd int, secondList []T,
	secondStart, secondEnd int, target []T, targetOffset int) {
	cursor1, cursor2 := firstStart, secondStart
	firstElement := firstList[cursor1]
	cursor1++
	secondElement := secondList[cursor2]
	cursor2++
	for {
		if compare(firstElement, secondElement) <= 0 {
			target[targetOffset] = firstElement
			targetOffset++
			if cursor1 == firstEnd {
				break
			}
			firstElement = firstList[cursor1]
			cursor1++
		} else {
			target[targetOffset] = secondElement
			targetOffset++
			if cursor2 != secondEnd {
				secondElement = secondList[cursor2]
				cursor2++
				continue
			}
			target[targetOffset] = firstElement
			targetOffset++
			setRange(target, targetOffset, targetOffset+(firstEnd-cursor1), firstList, cursor1)
			return
		}
	}
	target[targetOffset] = secondElement
	targetOffset++
	setRange(target, targetOffset, targetOffset+(secondEnd-cursor2), secondList, cursor2)
}
