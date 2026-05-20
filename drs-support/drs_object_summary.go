package drs_support

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cyverse/go-irodsclient/irods/common"
	"github.com/cyverse/go-irodsclient/irods/connection"
	"github.com/cyverse/go-irodsclient/irods/message"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
)

// DrsDataObjectSummary reports logical storage totals for data objects tagged as DRS objects.
type DrsDataObjectSummary struct {
	DataObjectCount int64 `json:"data_object_count"`
	TotalSize       int64 `json:"total_size"`
}

// DrsMetadataConnectionFilesystem exposes the low-level metadata connection needed for direct GenQuery.
type DrsMetadataConnectionFilesystem interface {
	GetMetadataConnection(allowShared bool) (*connection.IRODSConnection, error)
	ReturnMetadataConnection(conn *connection.IRODSConnection) error
}

// QueryDrsDataObjectSummary returns the logical count and cumulative size for data objects carrying
// a valid iRODS:DRS:ID AVU. It keeps the GenQuery condition to the DRS ID attribute name and applies
// DRS value/unit validation locally to match existing DRS lookup semantics without requiring OR
// support in base GenQuery.
func QueryDrsDataObjectSummary(filesystem DrsMetadataConnectionFilesystem) (summary DrsDataObjectSummary, err error) {
	if filesystem == nil {
		return summary, fmt.Errorf("missing iRODS filesystem")
	}

	conn, err := filesystem.GetMetadataConnection(true)
	if err != nil {
		return summary, fmt.Errorf("get metadata connection for DRS data object summary: %w", err)
	}
	defer func() {
		if returnErr := filesystem.ReturnMetadataConnection(conn); err == nil && returnErr != nil {
			err = fmt.Errorf("return metadata connection for DRS data object summary: %w", returnErr)
		}
	}()

	return queryDrsDataObjectSummaryWithConnection(conn)
}

func queryDrsDataObjectSummaryWithConnection(conn *connection.IRODSConnection) (DrsDataObjectSummary, error) {
	var summary DrsDataObjectSummary
	if conn == nil {
		return summary, fmt.Errorf("missing iRODS connection")
	}

	conn.Lock()
	defer conn.Unlock()

	seenDataIDs := map[int64]struct{}{}
	continueIndex := 0
	for {
		query := newDrsDataObjectSummaryQuery(conn, continueIndex)
		queryResult := message.IRODSMessageQueryResponse{}
		if err := conn.Request(query, &queryResult, nil, conn.GetOperationTimeout()); err != nil {
			if irodstypes.GetIRODSErrorCode(err) == common.CAT_NO_ROWS_FOUND {
				return summary, nil
			}
			return summary, fmt.Errorf("query DRS data object summary: %w", err)
		}
		if err := queryResult.CheckError(); err != nil {
			if irodstypes.GetIRODSErrorCode(err) == common.CAT_NO_ROWS_FOUND {
				return summary, nil
			}
			return summary, fmt.Errorf("query DRS data object summary: %w", err)
		}

		if err := appendDrsDataObjectSummaryRows(&summary, seenDataIDs, &queryResult); err != nil {
			return summary, err
		}

		if queryResult.ContinueIndex == 0 {
			break
		}
		continueIndex = queryResult.ContinueIndex
	}

	return summary, nil
}

func newDrsDataObjectSummaryQuery(conn *connection.IRODSConnection, continueIndex int) *message.IRODSMessageQueryRequest {
	query := message.NewIRODSMessageQueryRequest(common.MaxQueryRows, continueIndex, 0, 0)
	query.AddKeyVal(common.ZONE_KW, conn.GetAccount().ClientZone)
	query.AddSelect(common.ICAT_COLUMN_D_DATA_ID, 1)
	query.AddSelect(common.ICAT_COLUMN_DATA_SIZE, 1)
	query.AddSelect(common.ICAT_COLUMN_META_DATA_ATTR_VALUE, 1)
	query.AddSelect(common.ICAT_COLUMN_META_DATA_ATTR_UNITS, 1)
	query.AddEqualStringCondition(common.ICAT_COLUMN_META_DATA_ATTR_NAME, DrsIdAvuAttrib)
	return query
}

func appendDrsDataObjectSummaryRows(summary *DrsDataObjectSummary, seenDataIDs map[int64]struct{}, queryResult *message.IRODSMessageQueryResponse) error {
	if summary == nil {
		return fmt.Errorf("missing DRS data object summary")
	}
	if seenDataIDs == nil {
		return fmt.Errorf("missing DRS data object deduplication state")
	}
	if queryResult == nil || queryResult.RowCount == 0 {
		return nil
	}

	dataIDValues, ok := sqlResultValues(queryResult, common.ICAT_COLUMN_D_DATA_ID)
	if !ok {
		return fmt.Errorf("DRS data object summary query result missing DATA_ID column")
	}
	sizeValues, ok := sqlResultValues(queryResult, common.ICAT_COLUMN_DATA_SIZE)
	if !ok {
		return fmt.Errorf("DRS data object summary query result missing DATA_SIZE column")
	}
	avuValueValues, ok := sqlResultValues(queryResult, common.ICAT_COLUMN_META_DATA_ATTR_VALUE)
	if !ok {
		return fmt.Errorf("DRS data object summary query result missing metadata value column")
	}
	avuUnitValues, ok := sqlResultValues(queryResult, common.ICAT_COLUMN_META_DATA_ATTR_UNITS)
	if !ok {
		return fmt.Errorf("DRS data object summary query result missing metadata unit column")
	}

	for row := 0; row < queryResult.RowCount; row++ {
		avuValue, err := sqlResultRowValue(avuValueValues, row, "META_DATA_ATTR_VALUE")
		if err != nil {
			return err
		}
		avuUnits, err := sqlResultRowValue(avuUnitValues, row, "META_DATA_ATTR_UNITS")
		if err != nil {
			return err
		}
		if !isValidDrsIDAVUValue(avuValue, avuUnits) {
			continue
		}

		dataIDValue, err := sqlResultRowValue(dataIDValues, row, "DATA_ID")
		if err != nil {
			return err
		}
		dataID, err := parseCatalogInt64("DATA_ID", dataIDValue)
		if err != nil {
			return err
		}
		if _, seen := seenDataIDs[dataID]; seen {
			continue
		}

		sizeValue, err := sqlResultRowValue(sizeValues, row, "DATA_SIZE")
		if err != nil {
			return err
		}
		size, err := parseCatalogInt64("DATA_SIZE", sizeValue)
		if err != nil {
			return err
		}

		seenDataIDs[dataID] = struct{}{}
		summary.DataObjectCount++
		summary.TotalSize += size
	}

	return nil
}

func isValidDrsIDAVUValue(value string, units string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	return strings.TrimSpace(units) == "" || strings.EqualFold(strings.TrimSpace(units), DrsAvuUnit)
}

func sqlResultValues(queryResult *message.IRODSMessageQueryResponse, column common.ICATColumnNumber) ([]string, bool) {
	if queryResult == nil {
		return nil, false
	}
	for idx := range queryResult.SQLResult {
		if queryResult.SQLResult[idx].AttributeIndex == int(column) {
			return queryResult.SQLResult[idx].Values, true
		}
	}
	return nil, false
}

func sqlResultRowValue(values []string, row int, columnName string) (string, error) {
	if row < 0 || row >= len(values) {
		return "", fmt.Errorf("DRS data object summary query result missing %s value for row %d", columnName, row)
	}
	return values[row], nil
}

func parseCatalogInt64(columnName string, value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse DRS data object summary %s %q: %w", columnName, value, err)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("parse DRS data object summary %s %q: negative values are not supported", columnName, value)
	}
	return parsed, nil
}
