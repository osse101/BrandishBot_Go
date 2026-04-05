1. Add `regions_test.go` to test logic in `regions.go` to cover `LoadSearchRegions`, `resolveRegion`, and `rollRegionItemDrop`
   - Test `LoadSearchRegions` using a temporary file.
   - Test `resolveRegion` using different `explorerLevel` and `itemHint`.
   - Test `rollRegionItemDrop` using statistical distribution or fixed random seed if possible.
2. Complete pre commit steps to ensure proper testing, verification, review, and reflection are done.
3. Submit the change with a descriptive commit message.
