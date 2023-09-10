package netquery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	"github.com/khulnasoft-lab/portmaster/netquery/orm"
	"github.com/safing/portbase/log"
)

var charOnlyRegexp = regexp.MustCompile("[a-zA-Z]+")

type (

	// QueryHandler implements http.Handler and allows to perform SQL
	// query and aggregate functions on Database.
	QueryHandler struct {
		IsDevMode func() bool
		Database  *Database
	}
)

func (qh *QueryHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	start := time.Now()
	requestPayload, err := qh.parseRequest(req)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)

		return
	}

	queryParsed := time.Since(start)

	query, paramMap, err := requestPayload.generateSQL(req.Context(), qh.Database.Schema)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusBadRequest)

		return
	}

	sqlQueryBuilt := time.Since(start)

	// actually execute the query against the database and collect the result
	var result []map[string]interface{}
	if err := qh.Database.Execute(
		req.Context(),
		query,
		orm.WithNamedArgs(paramMap),
		orm.WithResult(&result),
		orm.WithSchema(*qh.Database.Schema),
	); err != nil {
		http.Error(resp, "Failed to execute query: "+err.Error(), http.StatusInternalServerError)

		return
	}
	sqlQueryFinished := time.Since(start)

	// send the HTTP status code
	resp.WriteHeader(http.StatusOK)

	// prepare the result encoder.
	enc := json.NewEncoder(resp)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")

	// prepare the result body that, in dev mode, contains
	// some diagnostics data about the query
	var resultBody map[string]interface{}
	if qh.IsDevMode() {
		resultBody = map[string]interface{}{
			"sql_prep_stmt": query,
			"sql_params":    paramMap,
			"query":         requestPayload.Query,
			"orderBy":       requestPayload.OrderBy,
			"groupBy":       requestPayload.GroupBy,
			"selects":       requestPayload.Select,
			"times": map[string]interface{}{
				"start_time":           start,
				"query_parsed_after":   queryParsed.String(),
				"query_built_after":    sqlQueryBuilt.String(),
				"query_executed_after": sqlQueryFinished.String(),
			},
		}
	} else {
		resultBody = make(map[string]interface{})
	}
	resultBody["results"] = result

	// and finally stream the response
	if err := enc.Encode(resultBody); err != nil {
		// we failed to encode the JSON body to resp so we likely either already sent a
		// few bytes or the pipe was already closed. In either case, trying to send the
		// error using http.Error() is non-sense. We just log it out here and that's all
		// we can do.
		log.Errorf("failed to encode JSON response: %s", err)

		return
	}
}

func (qh *QueryHandler) parseRequest(req *http.Request) (*QueryRequestPayload, error) { //nolint:dupl
	var body io.Reader

	switch req.Method {
	case http.MethodPost, http.MethodPut:
		body = req.Body
	case http.MethodGet:
		body = strings.NewReader(req.URL.Query().Get("q"))
	default:
		return nil, fmt.Errorf("invalid HTTP method")
	}

	var requestPayload QueryRequestPayload
	blob, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body" + err.Error())
	}

	body = bytes.NewReader(blob)

	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	if err := json.Unmarshal(blob, &requestPayload); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	return &requestPayload, nil
}

func (req *QueryRequestPayload) generateSQL(ctx context.Context, schema *orm.TableSchema) (string, map[string]interface{}, error) {
	if err := req.prepareSelectedFields(ctx, schema); err != nil {
		return "", nil, fmt.Errorf("perparing selected fields: %w", err)
	}

	// build the SQL where clause from the payload query
	whereClause, paramMap, err := req.Query.toSQLWhereClause(
		ctx,
		"",
		schema,
		orm.DefaultEncodeConfig,
	)
	if err != nil {
		return "", nil, fmt.Errorf("generating where clause: %w", err)
	}

	req.mergeParams(paramMap)

	if req.TextSearch != nil {
		textClause, textParams, err := req.TextSearch.toSQLConditionClause(ctx, schema, "", orm.DefaultEncodeConfig)
		if err != nil {
			return "", nil, fmt.Errorf("generating text-search clause: %w", err)
		}

		if textClause != "" {
			if whereClause != "" {
				whereClause += " AND "
			}

			whereClause += textClause

			req.mergeParams(textParams)
		}
	}

	groupByClause, err := req.generateGroupByClause(schema)
	if err != nil {
		return "", nil, fmt.Errorf("generating group-by clause: %w", err)
	}

	orderByClause, err := req.generateOrderByClause(schema)
	if err != nil {
		return "", nil, fmt.Errorf("generating order-by clause: %w", err)
	}

	selectClause := req.generateSelectClause()

	if whereClause != "" {
		whereClause = "WHERE " + whereClause
	}

	// if no database is specified we default to LiveDatabase only.
	if len(req.Databases) == 0 {
		req.Databases = []DatabaseName{LiveDatabase}
	}

	sources := make([]string, len(req.Databases))
	for idx, db := range req.Databases {
		sources[idx] = fmt.Sprintf("SELECT * FROM %s.connections %s", db, whereClause)
	}

	source := strings.Join(sources, " UNION ")

	query := `SELECT ` + selectClause + ` FROM ( ` + source + ` ) `

	query += " " + groupByClause + " " + orderByClause + " " + req.Pagination.toSQLLimitOffsetClause()

	return strings.TrimSpace(query), req.paramMap, nil
}

func (req *QueryRequestPayload) prepareSelectedFields(ctx context.Context, schema *orm.TableSchema) error {
	for idx, s := range req.Select {
		var field string

		switch {
		case s.Count != nil:
			field = s.Count.Field
		case s.Distinct != nil:
			field = *s.Distinct
		case s.Sum != nil:
			if s.Sum.Field != "" {
				field = s.Sum.Field
			} else {
				field = "*"
			}
		case s.Min != nil:
			if s.Min.Field != "" {
				field = s.Min.Field
			} else {
				field = "*"
			}
		default:
			field = s.Field
		}

		colName := "*"
		if field != "*" || (s.Count == nil && s.Sum == nil) {
			var err error

			colName, err = req.validateColumnName(schema, field)
			if err != nil {
				return err
			}
		}

		switch {
		case s.Count != nil:
			as := s.Count.As
			if as == "" {
				as = fmt.Sprintf("%s_count", colName)
			}
			distinct := ""
			if s.Count.Distinct {
				distinct = "DISTINCT "
			}
			req.selectedFields = append(
				req.selectedFields,
				fmt.Sprintf("COUNT(%s%s) AS %s", distinct, colName, as),
			)
			req.whitelistedFields = append(req.whitelistedFields, as)

		case s.Sum != nil:
			if s.Sum.As == "" {
				return fmt.Errorf("missing 'as' for $sum")
			}

			var (
				clause string
				params map[string]any
			)

			if s.Sum.Field != "" {
				clause = s.Sum.Field
			} else {
				var err error
				clause, params, err = s.Sum.Condition.toSQLWhereClause(ctx, fmt.Sprintf("sel%d", idx), schema, orm.DefaultEncodeConfig)
				if err != nil {
					return fmt.Errorf("in $sum: %w", err)
				}
			}

			req.mergeParams(params)
			req.selectedFields = append(
				req.selectedFields,
				fmt.Sprintf("SUM(%s) AS %s", clause, s.Sum.As),
			)
			req.whitelistedFields = append(req.whitelistedFields, s.Sum.As)

		case s.Min != nil:
			if s.Min.As == "" {
				return fmt.Errorf("missing 'as' for $min")
			}

			var (
				clause string
				params map[string]any
			)

			if s.Min.Field != "" {
				clause = field
			} else {
				var err error
				clause, params, err = s.Min.Condition.toSQLWhereClause(ctx, fmt.Sprintf("sel%d", idx), schema, orm.DefaultEncodeConfig)
				if err != nil {
					return fmt.Errorf("in $min: %w", err)
				}
			}

			req.mergeParams(params)
			req.selectedFields = append(
				req.selectedFields,
				fmt.Sprintf("MIN(%s) AS %s", clause, s.Min.As),
			)
			req.whitelistedFields = append(req.whitelistedFields, s.Min.As)

		case s.Distinct != nil:
			req.selectedFields = append(req.selectedFields, fmt.Sprintf("DISTINCT %s", colName))
			req.whitelistedFields = append(req.whitelistedFields, colName)

		default:
			req.selectedFields = append(req.selectedFields, colName)
		}
	}

	return nil
}

func (req *QueryRequestPayload) mergeParams(params map[string]any) {
	if req.paramMap == nil {
		req.paramMap = make(map[string]any)
	}

	for key, value := range params {
		req.paramMap[key] = value
	}
}

func (req *QueryRequestPayload) generateGroupByClause(schema *orm.TableSchema) (string, error) {
	if len(req.GroupBy) == 0 {
		return "", nil
	}

	groupBys := make([]string, len(req.GroupBy))
	for idx, name := range req.GroupBy {
		colName, err := req.validateColumnName(schema, name)
		if err != nil {
			return "", err
		}

		groupBys[idx] = colName
	}
	groupByClause := "GROUP BY " + strings.Join(groupBys, ", ")

	// if there are no explicitly selected fields we default to the
	// group-by columns as that's what's expected most of the time anyway...
	if len(req.selectedFields) == 0 {
		req.selectedFields = append(req.selectedFields, groupBys...)
	}

	return groupByClause, nil
}

func (req *QueryRequestPayload) generateSelectClause() string {
	selectClause := "*"
	if len(req.selectedFields) > 0 {
		selectClause = strings.Join(req.selectedFields, ", ")
	}

	return selectClause
}

func (req *QueryRequestPayload) generateOrderByClause(schema *orm.TableSchema) (string, error) {
	if len(req.OrderBy) == 0 {
		return "", nil
	}

	orderBys := make([]string, len(req.OrderBy))
	for idx, sort := range req.OrderBy {
		colName, err := req.validateColumnName(schema, sort.Field)
		if err != nil {
			return "", err
		}

		if sort.Desc {
			orderBys[idx] = fmt.Sprintf("%s DESC", colName)
		} else {
			orderBys[idx] = fmt.Sprintf("%s ASC", colName)
		}
	}

	return "ORDER BY " + strings.Join(orderBys, ", "), nil
}

func (req *QueryRequestPayload) validateColumnName(schema *orm.TableSchema, field string) (string, error) {
	colDef := schema.GetColumnDef(field)
	if colDef != nil {
		return colDef.Name, nil
	}

	if slices.Contains(req.whitelistedFields, field) {
		return field, nil
	}

	if slices.Contains(req.selectedFields, field) {
		return field, nil
	}

	return "", fmt.Errorf("column name %q not allowed", field)
}

// Compile time check.
var _ http.Handler = new(QueryHandler)
