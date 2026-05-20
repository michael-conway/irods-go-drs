package drs_support

import (
	"strings"
	"testing"

	"github.com/cyverse/go-irodsclient/irods/common"
	"github.com/cyverse/go-irodsclient/irods/connection"
	"github.com/cyverse/go-irodsclient/irods/message"
	irodstypes "github.com/cyverse/go-irodsclient/irods/types"
)

func TestAppendDrsDataObjectSummaryRowsDeduplicatesLogicalDataObjects(t *testing.T) {
	queryResult := &message.IRODSMessageQueryResponse{
		RowCount: 6,
		SQLResult: []message.IRODSMessageSQLResult{
			{
				AttributeIndex: int(common.ICAT_COLUMN_D_DATA_ID),
				Values:         []string{"101", "101", "102", "103", "104", "105"},
			},
			{
				AttributeIndex: int(common.ICAT_COLUMN_DATA_SIZE),
				Values:         []string{"7", "7", "11", "0", "17", "19"},
			},
			{
				AttributeIndex: int(common.ICAT_COLUMN_META_DATA_ATTR_VALUE),
				Values:         []string{"drs-a", "drs-a", "drs-b", "drs-c", "", "drs-wrong-unit"},
			},
			{
				AttributeIndex: int(common.ICAT_COLUMN_META_DATA_ATTR_UNITS),
				Values:         []string{DrsAvuUnit, DrsAvuUnit, "", DrsAvuUnit, DrsAvuUnit, "other"},
			},
		},
	}

	summary := DrsDataObjectSummary{}
	seen := map[int64]struct{}{}
	if err := appendDrsDataObjectSummaryRows(&summary, seen, queryResult); err != nil {
		t.Fatalf("append DRS data object summary rows: %v", err)
	}

	if summary.DataObjectCount != 3 {
		t.Fatalf("expected 3 logical data objects, got %d", summary.DataObjectCount)
	}
	if summary.TotalSize != 18 {
		t.Fatalf("expected total size 18, got %d", summary.TotalSize)
	}
}

func TestAppendDrsDataObjectSummaryRowsCarriesDeduplicationAcrossPages(t *testing.T) {
	summary := DrsDataObjectSummary{}
	seen := map[int64]struct{}{}

	firstPage := &message.IRODSMessageQueryResponse{
		RowCount: 1,
		SQLResult: []message.IRODSMessageSQLResult{
			{AttributeIndex: int(common.ICAT_COLUMN_D_DATA_ID), Values: []string{"101"}},
			{AttributeIndex: int(common.ICAT_COLUMN_DATA_SIZE), Values: []string{"7"}},
			{AttributeIndex: int(common.ICAT_COLUMN_META_DATA_ATTR_VALUE), Values: []string{"drs-a"}},
			{AttributeIndex: int(common.ICAT_COLUMN_META_DATA_ATTR_UNITS), Values: []string{DrsAvuUnit}},
		},
	}
	secondPage := &message.IRODSMessageQueryResponse{
		RowCount: 2,
		SQLResult: []message.IRODSMessageSQLResult{
			{AttributeIndex: int(common.ICAT_COLUMN_D_DATA_ID), Values: []string{"101", "102"}},
			{AttributeIndex: int(common.ICAT_COLUMN_DATA_SIZE), Values: []string{"7", "11"}},
			{AttributeIndex: int(common.ICAT_COLUMN_META_DATA_ATTR_VALUE), Values: []string{"drs-a", "drs-b"}},
			{AttributeIndex: int(common.ICAT_COLUMN_META_DATA_ATTR_UNITS), Values: []string{DrsAvuUnit, DrsAvuUnit}},
		},
	}

	if err := appendDrsDataObjectSummaryRows(&summary, seen, firstPage); err != nil {
		t.Fatalf("append first page: %v", err)
	}
	if err := appendDrsDataObjectSummaryRows(&summary, seen, secondPage); err != nil {
		t.Fatalf("append second page: %v", err)
	}

	if summary.DataObjectCount != 2 {
		t.Fatalf("expected 2 logical data objects, got %d", summary.DataObjectCount)
	}
	if summary.TotalSize != 18 {
		t.Fatalf("expected total size 18, got %d", summary.TotalSize)
	}
}

func TestAppendDrsDataObjectSummaryRowsRejectsMalformedCatalogRows(t *testing.T) {
	queryResult := &message.IRODSMessageQueryResponse{
		RowCount: 1,
		SQLResult: []message.IRODSMessageSQLResult{
			{AttributeIndex: int(common.ICAT_COLUMN_D_DATA_ID), Values: []string{"101"}},
			{AttributeIndex: int(common.ICAT_COLUMN_DATA_SIZE), Values: []string{"not-a-size"}},
			{AttributeIndex: int(common.ICAT_COLUMN_META_DATA_ATTR_VALUE), Values: []string{"drs-a"}},
			{AttributeIndex: int(common.ICAT_COLUMN_META_DATA_ATTR_UNITS), Values: []string{DrsAvuUnit}},
		},
	}

	err := appendDrsDataObjectSummaryRows(&DrsDataObjectSummary{}, map[int64]struct{}{}, queryResult)
	if err == nil {
		t.Fatal("expected malformed DATA_SIZE to fail")
	}
	if !strings.Contains(err.Error(), "DATA_SIZE") {
		t.Fatalf("expected DATA_SIZE error, got %v", err)
	}
}

func TestNewDrsDataObjectSummaryQueryMatchesDRSIDAttributeOnly(t *testing.T) {
	account, err := irodstypes.CreateIRODSAccount("localhost", 1247, "rods", "tempZone", irodstypes.AuthSchemeNative, "rods", "demoResc")
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	conn, err := connection.NewIRODSConnection(account, nil)
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}

	query := newDrsDataObjectSummaryQuery(conn, 17)

	if query.MaxRows != common.MaxQueryRows {
		t.Fatalf("expected max rows %d, got %d", common.MaxQueryRows, query.MaxRows)
	}
	if query.ContinueIndex != 17 {
		t.Fatalf("expected continue index 17, got %d", query.ContinueIndex)
	}
	assertQuerySelect(t, query, common.ICAT_COLUMN_D_DATA_ID)
	assertQuerySelect(t, query, common.ICAT_COLUMN_DATA_SIZE)
	assertQuerySelect(t, query, common.ICAT_COLUMN_META_DATA_ATTR_VALUE)
	assertQuerySelect(t, query, common.ICAT_COLUMN_META_DATA_ATTR_UNITS)
	assertQueryCondition(t, query, common.ICAT_COLUMN_META_DATA_ATTR_NAME, "= &#39;"+DrsIdAvuAttrib+"&#39;")
	assertQueryDoesNotCondition(t, query, common.ICAT_COLUMN_META_DATA_ATTR_UNITS)
}

func assertQuerySelect(t *testing.T, query *message.IRODSMessageQueryRequest, column common.ICATColumnNumber) {
	t.Helper()
	for idx, key := range query.Selects.Keys {
		if key == int(column) {
			if query.Selects.Values[idx] != 1 {
				t.Fatalf("expected select value 1 for column %d, got %d", column, query.Selects.Values[idx])
			}
			return
		}
	}
	t.Fatalf("expected selected column %d", column)
}

func assertQueryCondition(t *testing.T, query *message.IRODSMessageQueryRequest, column common.ICATColumnNumber, expected string) {
	t.Helper()
	for idx, key := range query.Conditions.Keys {
		if key == int(column) {
			if query.Conditions.Values[idx].Value != expected {
				t.Fatalf("expected condition %q for column %d, got %q", expected, column, query.Conditions.Values[idx].Value)
			}
			return
		}
	}
	t.Fatalf("expected condition for column %d", column)
}

func assertQueryDoesNotCondition(t *testing.T, query *message.IRODSMessageQueryRequest, column common.ICATColumnNumber) {
	t.Helper()
	for _, key := range query.Conditions.Keys {
		if key == int(column) {
			t.Fatalf("did not expect condition for column %d", column)
		}
	}
}
