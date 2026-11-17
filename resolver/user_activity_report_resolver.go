package resolvers

import (
	"cms_dcdd_user_management_service/helpers"
	"cms_dcdd_user_management_service/model"

	"fmt"
	"time"

	"github.com/graphql-go/graphql"
)

func (ur *UserResolver) FetchUserActivityReport(p graphql.ResolveParams) *model.DcddGenericUserResponse {
	filter := model.UserActivityFilter{Limit: 20}
	filter.Activity, _ = p.Args["activity"].(string)
	filter.SearchBy, _ = p.Args["search_by"].(string)
	filter.Search, _ = p.Args["search"].(string)
	if l, ok := p.Args["limit"].(int); ok && l > 0 {
		filter.Limit = l
	}
	if o, ok := p.Args["offset"].(int); ok && o > 0 {
		filter.Offset = o
	}

	var err error
	if filter.FromDate, err = parseReportDate(p.Args["from_date"], "from_date"); err != nil {
		return helpers.FormatError(err)
	}
	if filter.ToDate, err = parseReportDate(p.Args["to_date"], "to_date"); err != nil {
		return helpers.FormatError(err)
	}

	users, total, err := ur.Services.FetchUserActivityReport(filter)
	if err != nil {
		return helpers.FormatError(err)
	}

	return &model.DcddGenericUserResponse{
		Data: &model.UserActivityReportResult{
			Users: users,
			Total: total,
			Pagination: &model.DcddUserPagination{
				CurrentPage: filter.Offset/filter.Limit + 1,
				TotalPage:   (total + filter.Limit - 1) / filter.Limit,
				Limit:       filter.Limit,
			},
		},
	}
}

// Report dates are calendar days as admins in Bhutan see them (UTC+6, no daylight saving).
var reportTimezone = time.FixedZone("BTT", 6*60*60)

// parseReportDate reads an optional YYYY-MM-DD argument as midnight Bhutan time.
func parseReportDate(arg interface{}, name string) (*time.Time, error) {
	s, _ := arg.(string)
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, reportTimezone)
	if err != nil {
		return nil, fmt.Errorf("invalid %s %q: use YYYY-MM-DD", name, s)
	}
	return &t, nil
}
