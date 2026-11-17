package repositories

import (
    "fmt"
    "time"

    "cms_dcdd_user_management_service/model"

    "gorm.io/gorm"
)

// FetchUserActivityReport lists DCDD users with their last login.
func (repo *UserRepository) FetchUserActivityReport(filter model.UserActivityFilter) ([]model.UserActivityReportRow, int, error) {
    query := repo.DB.Table("dcdd_auth.dcdd_users AS u").
        Joins("LEFT JOIN dcdd_user_data.dcdd_user_profiles p ON p.user_id = u.id").
        Joins("LEFT JOIN LATERAL (SELECT MAX(t.created_at) AS last_login_at FROM dcdd_auth.dcdd_access_tokens t WHERE t.user_id = u.id) l ON true")

    switch filter.Activity {
    case "", "all":
        if filter.FromDate != nil || filter.ToDate != nil {
            query = query.Where(loginWithin(filter.FromDate, filter.ToDate))
        }
    case "login":
        query = query.Where(loginWithin(filter.FromDate, filter.ToDate))
    default:
        return nil, 0, fmt.Errorf("invalid activity %q: DCDD users only support all or login", filter.Activity)
    }

    query, err := applyUserSearch(query, "u", filter.SearchBy, filter.Search)
    if err != nil {
        return nil, 0, err
    }

    var total int64
    if err := repo.DB.Table("(?) AS filtered", query.Session(&gorm.Session{}).Select("u.id")).Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count user activities: %v", err)
    }

    var rows []model.UserActivityReportRow
    err = query.Select("u.id AS user_id, p.name, p.cid, u.email, u.mobile_no, u.category, u.status, l.last_login_at").
        Order("l.last_login_at DESC NULLS LAST, u.created_at DESC").
        Limit(filter.Limit).Offset(filter.Offset).
        Scan(&rows).Error
    if err != nil {
        return nil, 0, fmt.Errorf("failed to fetch user activities: %v", err)
    }
    return rows, int(total), nil
}

// loginWithin matches users with a login in the inclusive date range; open ends are unbounded.
func loginWithin(from, to *time.Time) interface{} {
    start := time.Time{}
    end := time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
    if from != nil {
        start = *from
    }
    if to != nil {
        end = to.AddDate(0, 0, 1)
    }
    return gorm.Expr("EXISTS (SELECT 1 FROM dcdd_auth.dcdd_access_tokens t WHERE t.user_id = u.id AND t.created_at >= ? AND t.created_at < ?)", start, end)
}
