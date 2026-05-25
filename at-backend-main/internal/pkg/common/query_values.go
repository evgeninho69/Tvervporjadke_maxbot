package common

import (
	"net/url"
	"strconv"
)

const defaultPageSize = 10

func GetPageSize(query url.Values) uint64 {
	pageSizeString := query.Get("page_size")
	if pageSizeString == "" {
		return defaultPageSize
	}

	pageSize, err := strconv.ParseUint(pageSizeString, 10, 64)
	if err != nil {
		return defaultPageSize
	}

	return pageSize
}