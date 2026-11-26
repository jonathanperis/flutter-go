package dart

// ListSort is dart:core's List.sort(compare) as the Dart VM runs it:
// dart:_internal's Sort.sort (sdk/lib/internal/sort.dart), an insertion sort
// for ranges of at most 33 elements and Yaroslavskiy's dual-pivot quicksort
// above that. Neither is stable in general (the insertion sort is), so the
// order of elements that compare equal is observable, and Flutter code that
// sorts with ties (semantics traversal order, dirty-node depths) needs this
// exact algorithm to match Dart.
func ListSort[E any](a []E, compare func(a, b E) int) {
	sortDoSort(a, 0, len(a)-1, compare)
}

// ListSortRange is Sort.sortRange: it sorts a[from:to].
func ListSortRange[E any](a []E, from, to int, compare func(a, b E) int) {
	if from < 0 || to > len(a) || to < from {
		panic("OutOfRange")
	}
	sortDoSort(a, from, to-1, compare)
}

const sortInsertionSortThreshold = 32

func sortDoSort[E any](a []E, left, right int, compare func(a, b E) int) {
	if right-left <= sortInsertionSortThreshold {
		sortInsertionSort(a, left, right, compare)
	} else {
		sortDualPivotQuicksort(a, left, right, compare)
	}
}

func sortInsertionSort[E any](a []E, left, right int, compare func(a, b E) int) {
	for i := left + 1; i <= right; i++ {
		el := a[i]
		j := i
		for j > left && compare(a[j-1], el) > 0 {
			a[j] = a[j-1]
			j--
		}
		a[j] = el
	}
}

func sortDualPivotQuicksort[E any](a []E, left, right int, compare func(a, b E) int) {
	if right-left <= sortInsertionSortThreshold {
		panic(NewAssertionError(nil))
	}
	sixth := (right - left + 1) / 6
	index1 := left + sixth
	index5 := right - sixth
	index3 := (left + right) / 2 // The midpoint (left+right is never negative).
	index2 := index3 - sixth
	index4 := index3 + sixth

	el1, el2, el3, el4, el5 := a[index1], a[index2], a[index3], a[index4], a[index5]

	// Sort the selected 5 elements using a sorting network.
	if compare(el1, el2) > 0 {
		el1, el2 = el2, el1
	}
	if compare(el4, el5) > 0 {
		el4, el5 = el5, el4
	}
	if compare(el1, el3) > 0 {
		el1, el3 = el3, el1
	}
	if compare(el2, el3) > 0 {
		el2, el3 = el3, el2
	}
	if compare(el1, el4) > 0 {
		el1, el4 = el4, el1
	}
	if compare(el3, el4) > 0 {
		el3, el4 = el4, el3
	}
	if compare(el2, el5) > 0 {
		el2, el5 = el5, el2
	}
	if compare(el2, el3) > 0 {
		el2, el3 = el3, el2
	}
	if compare(el4, el5) > 0 {
		el4, el5 = el5, el4
	}

	pivot1 := el2
	pivot2 := el4

	// el2 and el4 have been saved in the pivot variables. They will be
	// written back, once the partitioning is finished.
	a[index1] = el1
	a[index3] = el3
	a[index5] = el5

	a[index2] = a[left]
	a[index4] = a[right]

	less := left + 1   // First element in the middle partition.
	great := right - 1 // Last element in the middle partition.

	pivotsAreEqual := compare(pivot1, pivot2) == 0
	if pivotsAreEqual {
		pivot := pivot1
		for k := less; k <= great; k++ {
			ak := a[k]
			comp := compare(ak, pivot)
			if comp == 0 {
				continue
			}
			if comp < 0 {
				if k != less {
					a[k] = a[less]
					a[less] = ak
				}
				less++
			} else {
				for {
					comp = compare(a[great], pivot)
					if comp > 0 {
						great--
						continue
					} else if comp < 0 {
						a[k] = a[less]
						a[less] = a[great]
						less++
						a[great] = ak
						great--
						break
					} else {
						a[k] = a[great]
						a[great] = ak
						great--
						break
					}
				}
			}
		}
	} else {
		for k := less; k <= great; k++ {
			ak := a[k]
			compPivot1 := compare(ak, pivot1)
			if compPivot1 < 0 {
				if k != less {
					a[k] = a[less]
					a[less] = ak
				}
				less++
			} else {
				compPivot2 := compare(ak, pivot2)
				if compPivot2 > 0 {
					for {
						comp := compare(a[great], pivot2)
						if comp > 0 {
							great--
							if great < k {
								break
							}
							continue
						} else {
							comp = compare(a[great], pivot1)
							if comp < 0 {
								a[k] = a[less]
								a[less] = a[great]
								less++
								a[great] = ak
								great--
							} else {
								a[k] = a[great]
								a[great] = ak
								great--
							}
							break
						}
					}
				}
			}
		}
	}

	// Move pivots into their final positions.
	a[left] = a[less-1]
	a[less-1] = pivot1
	a[right] = a[great+1]
	a[great+1] = pivot2

	sortDoSort(a, left, less-2, compare)
	sortDoSort(a, great+2, right, compare)

	if pivotsAreEqual {
		return
	}

	if less < index1 && great > index5 {
		for compare(a[less], pivot1) == 0 {
			less++
		}
		for compare(a[great], pivot2) == 0 {
			great--
		}
		for k := less; k <= great; k++ {
			ak := a[k]
			compPivot1 := compare(ak, pivot1)
			if compPivot1 == 0 {
				if k != less {
					a[k] = a[less]
					a[less] = ak
				}
				less++
			} else {
				compPivot2 := compare(ak, pivot2)
				if compPivot2 == 0 {
					for {
						comp := compare(a[great], pivot2)
						if comp == 0 {
							great--
							if great < k {
								break
							}
							continue
						} else {
							comp = compare(a[great], pivot1)
							if comp < 0 {
								a[k] = a[less]
								a[less] = a[great]
								less++
								a[great] = ak
								great--
							} else {
								a[k] = a[great]
								a[great] = ak
								great--
							}
							break
						}
					}
				}
			}
		}
		sortDoSort(a, less, great, compare)
	} else {
		sortDoSort(a, less, great, compare)
	}
}
