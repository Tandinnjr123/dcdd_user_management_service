package repositories

import (
    "dcdd_user_management_service/model"
    "dcdd_user_management_service/helpers"

    "fmt"
    "strings"
    "time"
    "errors"
    "context"

    "gorm.io/gorm"
	"github.com/google/uuid"
)

type UserRepository struct{
    DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
    return &UserRepository{DB: db}
}

func (repo *UserRepository) CheckForDcddExistingUser(field, value string) (*model.DcddUser, error) {
    var user model.DcddUser
    err := repo.DB.Where(fmt.Sprintf("%s = ? AND status != ?", field), value, "Deleted").First(&user).Error

    if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user with %v %v: %v", field, value, err)
	}
	return &user, nil
}

func (repo *UserRepository) FetchDcddUserByLoginID(field, value string) (*model.DcddUser, error) {
    var user model.DcddUser
    err := repo.DB.Where(fmt.Sprintf("%s = ?", field), value).First(&user).Error
    if err != nil {
        return nil, fmt.Errorf("failed to find user with %v %v", field, value)
    }
    return &user, nil
}

func (repo *UserRepository) CreateDcddUser(signupInput *model.SignupInput) (*model.DcddUser, *model.UserProfile, error) {
    var user *model.DcddUser
    var createdUser *model.DcddUser
    var userProfile *model.UserProfile
    var err error

    err = repo.DB.Transaction(func(tx *gorm.DB) error {
        identifier, err := helpers.GenerateRandomTokenString(6)
        if err != nil {
            return fmt.Errorf("failed to generate identifier: %v", err)
        }

        hashedPassword, err := helpers.EncryptPassword(signupInput.Password)
        if err != nil {
            return fmt.Errorf("failed to hash password: %v", err)
        }

        // Check for existing user by mobile and email within the transaction.
        var existingUser model.DcddUser
        if signupInput.MobileNo != "" {
            tx.Where("mobile_no = ?", signupInput.MobileNo).First(&existingUser)
        }
        if existingUser.ID == uuid.Nil && signupInput.Email != "" {
            tx.Where("email = ?", signupInput.Email).First(&existingUser)
        }
        user = &existingUser

        loginid := helpers.GenerateLoginId(6)

        // If a user is found, update their record.
        if user.ID != uuid.Nil {
            updateData := map[string]interface{}{
                "mobile_no":       signupInput.MobileNo,
                "email":           signupInput.Email,
                "user_identifier": identifier,
                "password":        hashedPassword,
                "status":          "Active",
                "updated_at":      time.Now(),
                "category":        signupInput.Category,
                "login_id":        loginid,
            }
            if err := tx.Model(user).Updates(updateData).Error; err != nil {
                return fmt.Errorf("failed to update user data: %v", err)
            }
            createdUser = user
            
        } else {
            // If no user is found, create a new one.
            newUser := model.DcddUser{
                ID:             uuid.New(),
                MobileNo:       helpers.StringPtr(signupInput.MobileNo),
                Email:          helpers.StringPtr(signupInput.Email),
                UserIdentifier: identifier,
                Password:       hashedPassword,
                Status:         "Active",
                CreatedAt:      time.Now(),
                UpdatedAt:      time.Now(),
                StudentId :     signupInput.StudentId,
                Category :      signupInput.Category,
                LoginId :       loginid,    
            }
            if err := tx.Create(&newUser).Error; err != nil {
                return fmt.Errorf("failed to insert user data: %v", err)
            }
            createdUser = &newUser
        }

        profileInput := model.UserProfileInput{
            UserId:     createdUser.ID,
            Gender:     signupInput.Gender,
            Name:       signupInput.Name,
            Cid:        signupInput.Cid,
            Dob:        signupInput.Dob,
            DzongkhagId: signupInput.DzongkhagId,
            EccdId:     signupInput.EccdId,
            SchoolId:   signupInput.SchoolId,
            GradeId:    signupInput.GradeId,
        }
        
        userProfile, err = repo.CreateDcddUserProfile(tx, profileInput)
        if err != nil {
            return fmt.Errorf("failed to create user profile: %v", err)
        }
        
        return nil
    })

    if err != nil {
        return nil, nil, err
    }

    return createdUser, userProfile, nil
}

func (repo *UserRepository) CreateDcddUserProfile(tx *gorm.DB, inputData model.UserProfileInput) (*model.UserProfile, error) {
    userProfile := model.UserProfile{
        ID:             uuid.New(),
        Name:           inputData.Name,
        Gender:         inputData.Gender,
        ProfilePicture: inputData.ProfilePicture,
        UserId:         inputData.UserId,
        GradeId:        inputData.GradeId,
        SchoolId:       inputData.SchoolId,
        EccdId:         inputData.EccdId,
        DzongkhagId:    inputData.DzongkhagId,
        Dob:            inputData.Dob,
        Cid:            inputData.Cid,
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }

    if err := tx.Create(&userProfile).Error; err != nil {
        return nil, fmt.Errorf("failed to insert user profile: %v", err)
    }

    if inputData.GradeId != uuid.Nil && inputData.GradeId != uuid.Nil {
       userProfile.GradeId = inputData.GradeId
   }
    if inputData.SchoolId != uuid.Nil && inputData.SchoolId != uuid.Nil {
       userProfile.SchoolId = inputData.SchoolId
   }
    if inputData.EccdId != uuid.Nil && inputData.EccdId != uuid.Nil {
       userProfile.EccdId = inputData.EccdId
   }

    return &userProfile, nil
}

func (repo *UserRepository) FetchProfileByDcddUserId(ctx context.Context, userId uuid.UUID) (*model.UserProfile, error) {
    var profile model.UserProfile

    db, err := helpers.GetGormDB()
    if err != nil {
        return nil, err
    }

    if err := db.WithContext(ctx).Where("user_id = ?", userId).First(&profile).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch profile for user id '%s': %w", userId, err)
    }
    return &profile, nil
}
// FetchDcddMembership returns the programme-wide DCDD membership, or nil when none is set up or
// the dcdd_membership tables are not in this service's database.
func (repo *UserRepository) FetchDcddMembership(ctx context.Context) (*model.DcddMembership, error) {
    var exists bool
    if err := repo.DB.WithContext(ctx).Raw("SELECT to_regclass('dcdd_membership.user_memberships') IS NOT NULL").Scan(&exists).Error; err != nil || !exists {
        return nil, nil
    }
    var memberships []model.DcddMembership
    err := repo.DB.WithContext(ctx).Table("dcdd_membership.user_memberships").
        Select("membership_duration_id, membership_join_date, membership_end_date, membership_end_date > now() AS is_active").
        Order("updated_at DESC").
        Limit(1).
        Scan(&memberships).Error
    if err != nil {
        return nil, fmt.Errorf("failed to fetch membership: %w", err)
    }
    if len(memberships) == 0 {
        return nil, nil
    }
    return &memberships[0], nil
}

func (repo *UserRepository) FetchAllDcddUsers(limit, offset int, searchBy, search string) ([]model.DcddUser, int, error) {
    var total int64
    var users []model.DcddUser

    filtered, err := applyUserSearch(repo.DB.Model(&model.DcddUser{}), "dcdd_auth.dcdd_users", searchBy, search)
    if err != nil {
        return nil, 0, err
    }

    if err := filtered.Session(&gorm.Session{}).Count(&total).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count users: %w", err)
    }

    db := filtered.Preload("UserProfile").
    Preload("UserProfile.Dzongkhag").
    Preload("UserProfile.School").
    Preload("UserProfile.Eccd").
    Preload("UserProfile.Grade").
    Limit(limit).Offset(offset)
    if err := db.Find(&users).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to fetch users: %v", err) 
    }

    return users, int(total), nil
}

func (repo *UserRepository) GetAllActiveDcddUsers(limit, offset int) ([]model.DcddUser, int, error){
    var users []model.DcddUser
	if err := repo.DB.Where("status = ?", "Active").
    Preload("UserProfile").
    Preload("UserProfile.Dzongkhag").
    Preload("UserProfile.School").
    Preload("UserProfile.Eccd").
    Preload("UserProfile.Grade").
    Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0,fmt.Errorf("failed to fetch users: %w", err)
	}

	var totalCount int64
    if err := repo.DB.Model(&model.DcddUser{}).Where("status = ?", "Active").Count(&totalCount).Error; err != nil {
        return nil, 0, fmt.Errorf("failed to count active users: %v", err)
    }

	return users, int(totalCount), nil
}

// FetchDcddUsersByDateRange lists users registered on the given days (inclusive, Bhutan time) with
// their profiles, newest first. Password hashes are cleared before returning.
func (repo *UserRepository) FetchDcddUsersByDateRange(fromDate, toDate time.Time) ([]model.DcddUserAndProfile, error) {
    r := newReportRange(&fromDate, &toDate)

    var users []model.DcddUser
    if err := repo.DB.
        Where("created_at >= ? AND created_at < ?", r.newFrom, r.to).
        Order("created_at DESC").
        Find(&users).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch users between %v and %v: %w", fromDate, toDate, err)
    }
    if len(users) == 0 {
        return []model.DcddUserAndProfile{}, nil
    }

    // Profiles are fetched in batches: Postgres allows at most 65535 parameters per query.
    const batchSize = 10000
    profileMap := make(map[uuid.UUID]*model.UserProfile, len(users))
    for start := 0; start < len(users); start += batchSize {
        end := min(start+batchSize, len(users))
        userIDs := make([]uuid.UUID, 0, end-start)
        for _, u := range users[start:end] {
            userIDs = append(userIDs, u.ID)
        }
        var profiles []model.UserProfile
        if err := repo.DB.Where("user_id IN ?", userIDs).Find(&profiles).Error; err != nil {
            return nil, fmt.Errorf("failed to fetch profiles: %w", err)
        }
        for i := range profiles {
            profileMap[profiles[i].UserId] = &profiles[i]
        }
    }

    results := make([]model.DcddUserAndProfile, len(users))
    for i := range users {
        users[i].Password = ""
        results[i] = model.DcddUserAndProfile{User: &users[i], UserProfile: profileMap[users[i].ID]}
    }
    return results, nil
}




func (repo *UserRepository) UpdateDcddUser(userID uuid.UUID, signupInput *model.SignupInput) (*model.DcddUser, *model.UserProfile, error) {
    var user model.DcddUser
    var userProfile model.UserProfile

    err := repo.DB.Transaction(func(tx *gorm.DB) error {
        if err := tx.First(&user, "id = ?", userID).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                return fmt.Errorf("user with ID %s not found: %w", userID, err)
            }
            return fmt.Errorf("failed to fetch user for update: %w", err)
        }
        updateData := map[string]interface{}{
            "updated_at": time.Now(),
        }

        if signupInput.Password != "" {
            hashedPassword, err := helpers.EncryptPassword(signupInput.Password)
            if err != nil {
                return fmt.Errorf("failed to hash password: %w", err)
            }
            updateData["password"] = hashedPassword
        }
        if signupInput.MobileNo != "" {
            updateData["mobile_no"] = signupInput.MobileNo
        }
        if signupInput.Email != "" {
            updateData["email"] = signupInput.Email
        }

        if err := tx.Model(&user).Updates(updateData).Error; err != nil {
            return fmt.Errorf("failed to update user data: %w", err)
        }

        if err := tx.Where("user_id = ?", user.ID).First(&userProfile).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                return fmt.Errorf("user profile not found for user ID %s: %w", user.ID, err)
            }
            return fmt.Errorf("failed to fetch user profile: %w", err)
        }

        profileUpdate := map[string]interface{}{}
        if signupInput.Name != "" {
            profileUpdate["name"] = signupInput.Name
        }
        if signupInput.Gender != "" {
            profileUpdate["gender"] = signupInput.Gender
        }
        if signupInput.Cid != "" {
            profileUpdate["cid"] = signupInput.Cid
        }
        if len(profileUpdate) > 0 {
            if err := tx.Model(&userProfile).Updates(profileUpdate).Error; err != nil {
                return fmt.Errorf("failed to update user profile: %w", err)
            }
        }

        return nil
    })

    if err != nil {
        return nil, nil, err
    }

    return &user, &userProfile, nil
}

func (repo *UserRepository) UpdateDcddUserPassword(userID uuid.UUID, DcddResetPasswordInput *model.SignupInput) (*model.DcddUser, *model.UserProfile, error) {
    var user model.DcddUser
    var userProfile model.UserProfile

    err := repo.DB.Transaction(func(tx *gorm.DB) error {
        if err := tx.First(&user, "id = ?", userID).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                return fmt.Errorf("user with ID %s not found: %w", userID, err)
            }
            return fmt.Errorf("failed to fetch user for update: %w", err)
        }
        updateData := map[string]interface{}{
            "updated_at": time.Now(),
        }

        if DcddResetPasswordInput.Password != "" {
            hashedPassword, err := helpers.EncryptPassword(DcddResetPasswordInput.Password)
            if err != nil {
                return fmt.Errorf("failed to hash password: %w", err)
            }
            updateData["password"] = hashedPassword
        }
        if err := tx.Model(&user).Updates(updateData).Error; err != nil {
            return fmt.Errorf("failed to update user data: %w", err)
        }

        if err := tx.Where("user_id = ?", user.ID).First(&userProfile).Error; err != nil {
            if errors.Is(err, gorm.ErrRecordNotFound) {
                return fmt.Errorf("user profile not found for user ID %s: %w", user.ID, err)
            }
            return fmt.Errorf("failed to fetch user profile: %w", err)
        }

        profileUpdate := map[string]interface{}{}
        if len(profileUpdate) > 0 {
            if err := tx.Model(&userProfile).Updates(profileUpdate).Error; err != nil {
                return fmt.Errorf("failed to update user profile: %w", err)
            }
        }
        return nil
    })

    if err != nil {
        return nil, nil, err
    }

    return &user, &userProfile, nil
}

func (repo *UserRepository) UpdateDcddUserStatus(ctx context.Context, userID uuid.UUID, status string) (*model.DcddUser, error) {
	// Find the existing question by its ID
	var user model.DcddUser
	if err := repo.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	user.Status = status
	if err := repo.DB.Save(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (repo *UserRepository) BulkRegistration(signupInputs []model.SignupInput) (error) {
    var users []*model.DcddUser
    var userProfiles []*model.UserProfile

    // Start a single transaction for the entire batch.
    err := repo.DB.Transaction(func(tx *gorm.DB) error {
        for _, signupInput := range signupInputs {
            identifier, err := helpers.GenerateRandomTokenString(6)
            if err != nil {
                return fmt.Errorf("failed to generate identifier: %v", err)
            }

            hashedPassword, err := helpers.EncryptPassword(signupInput.Password)
            if err != nil {
                return fmt.Errorf("failed to hash password: %v", err)
            }

            loginid := helpers.GenerateLoginId(6)

            newUser := model.DcddUser{
                ID:             uuid.New(),
                MobileNo:       helpers.StringPtr(signupInput.MobileNo),
                Email:          helpers.StringPtr(signupInput.Email),
                UserIdentifier: identifier,
                Password:       hashedPassword,
                Status:         "Active",
                CreatedAt:      time.Now(),
                UpdatedAt:      time.Now(),
                StudentId:     signupInput.StudentId,
                Category:      signupInput.Category,
                LoginId:       loginid,
            }

            if err := tx.Create(&newUser).Error; err != nil {
                return fmt.Errorf("failed to insert user data for '%s': %v", signupInput.Name, err)
            }
            users = append(users, &newUser)

            profileInput := model.UserProfileInput{
                UserId: newUser.ID,
                Gender: signupInput.Gender,
                Name:   signupInput.Name,
                MobileNo: signupInput.MobileNo,
                Email:    signupInput.Email,
                GradeId:  signupInput.GradeId,
                StudentId: signupInput.StudentId,
                Password:  signupInput.Password,
                SchoolId:  signupInput.SchoolId,
                EccdId:    signupInput.EccdId,
                Dob:       signupInput.Dob,
                Cid:       signupInput.Cid,
                DzongkhagId: signupInput.DzongkhagId,
                Category:signupInput.Category,
            }
            userProfile, err := repo.CreateDcddUserProfile(tx, profileInput)
            if err != nil {
                return fmt.Errorf("failed to create user profile for user '%s': %v", err)
            }
            userProfiles = append(userProfiles, userProfile)
        }
        return nil
    })
    return err
}

func (repo *UserRepository) GetSchool(ctx context.Context, id uuid.UUID) (*model.School, error) {
	db, err := helpers.GetGormDB()
    if err != nil {
        return nil, err
    }

	var school model.School
	if err := db.WithContext(ctx).Where("id = ?", id).First(&school).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch school: %v", err)
	}
	return &school, nil
}


func (repo *UserRepository) GetEccd(ctx context.Context, id uuid.UUID) (*model.Eccd, error) {
	db, err := helpers.GetGormDB()
    if err != nil {
        return nil, err
    }

	var eccd model.Eccd
	if err := db.WithContext(ctx).Where("id = ?", id).First(&eccd).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch eccd: %v", err)
	}
	return &eccd, nil
}

func (repo *UserRepository) GetGrade(ctx context.Context, id uuid.UUID) (*model.Grade, error) {
	db, err := helpers.GetGormDB()
    if err != nil {
        return nil, err
    }

	var grade model.Grade
	if err := db.WithContext(ctx).Where("id = ?", id).First(&grade).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch grade: %v", err)
	}
	return &grade, nil
}

func (repo *UserRepository) GetDzongkhag(ctx context.Context, id uuid.UUID) (*model.Dzongkhag, error) {
	db, err := helpers.GetGormDB()
    if err != nil {
        return nil, err
    }

	var dzongkhag model.Dzongkhag
	if err := db.WithContext(ctx).Where("id = ?", id).First(&dzongkhag).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch dzongkhag: %v", err)
	}
	return &dzongkhag, nil
}

func (repo *UserRepository) FetchDzongkhag(ctx context.Context) ([]model.Dzongkhag, error) {
	db, err := helpers.GetGormDB()
    if err != nil {
        return nil, err
    }

	var dzongkhags []model.Dzongkhag
	if err := db.WithContext(ctx).Find(&dzongkhags).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch dzongkhags: %v", err)
	}
	return dzongkhags, nil
}

func (repo *UserRepository) FetchGrade(ctx context.Context) ([]model.Grade, error) {
	db, err := helpers.GetGormDB()
    if err != nil {
        return nil, err
    }

	var grades []model.Grade
	if err := db.WithContext(ctx).Find(&grades).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %v", err)
	}
	return grades, nil
}

func (repo *UserRepository) FetchSchool(ctx context.Context, dzongkhagId uuid.UUID) ([]model.School, error) {
	db, err := helpers.GetGormDB()
    if err != nil {
        return nil, err
    }
	var schools []model.School
	if err := db.WithContext(ctx).Where("dzongkhag_id = ? ", dzongkhagId).Find(&schools).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch schools: %v", err)
	}
	return schools, nil
}


func (repo *UserRepository) FetchEccd(ctx context.Context, DzonghkhagId uuid.UUID) ([]model.Eccd, error) {
	db, err := helpers.GetGormDB()
    if err != nil {
        return nil, err
    }

	var eccds []model.Eccd
	if err := db.WithContext(ctx).Where("dzongkhag_id = ? ", DzonghkhagId).Find(&eccds).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch eccds: %v", err)
	}
	return eccds, nil
}

// GetDcddUserTotals counts DCDD users for the report cards, narrowed by dzongkhag, school, ECCD and
// category. Total is everyone registered by the end of the range; active is those of them who
// logged in or used the app in the range (any time without a range); inactive is the rest; new is
// those registered in the range, or in the last 30 days without one. App activity is recorded per
// month, so it counts for the whole month. Returns total, active, new, inactive.
func (repo *UserRepository) GetDcddUserTotals(fromDate, toDate *time.Time, school_id, eccd_id, dzongkhag_id *uuid.UUID, category *string) (totalAll int, totalActive int, totalNew int, totalInActive int, err error) {
    r := newReportRange(fromDate, toDate)
    args := map[string]interface{}{"active_from": r.activeFrom, "new_from": r.newFrom, "to": r.to}

    filters := ""
    profileFilters := []string{}
    if dzongkhag_id != nil {
        profileFilters = append(profileFilters, "up.dzongkhag_id = @dzongkhag_id")
        args["dzongkhag_id"] = *dzongkhag_id
    }
    if school_id != nil {
        profileFilters = append(profileFilters, "up.school_id = @school_id")
        args["school_id"] = *school_id
    }
    if eccd_id != nil {
        profileFilters = append(profileFilters, "up.eccd_id = @eccd_id")
        args["eccd_id"] = *eccd_id
    }
    if len(profileFilters) > 0 {
        filters += " AND EXISTS (SELECT 1 FROM dcdd_user_data.dcdd_user_profiles up WHERE up.user_id = u.id AND " +
            strings.Join(profileFilters, " AND ") + ")"
    }
    if category != nil {
        filters += " AND u.category = @category"
        args["category"] = *category
    }

    var counts struct {
        TotalAll    int
        TotalActive int
        TotalNew    int
    }
    err = repo.DB.Raw(`
        SELECT COUNT(*) AS total_all,
            COUNT(*) FILTER (WHERE
                EXISTS (SELECT 1 FROM dcdd_auth.dcdd_access_tokens t
                    WHERE t.user_id = u.id AND t.created_at >= @active_from AND t.created_at < @to)
                OR EXISTS (SELECT 1 FROM dcdd_auth.dcdd_user_activities a
                    WHERE a.user_id = u.id
                    AND (make_date(a.year, a.month, 1) + interval '1 month') AT TIME ZONE 'Asia/Thimphu' > @active_from
                    AND make_date(a.year, a.month, 1)::timestamp AT TIME ZONE 'Asia/Thimphu' < @to)
            ) AS total_active,
            COUNT(*) FILTER (WHERE u.created_at >= @new_from AND u.created_at < @to) AS total_new
        FROM dcdd_auth.dcdd_users u
        WHERE u.created_at < @to`+filters, args).
        Scan(&counts).Error
    if err != nil {
        return 0, 0, 0, 0, fmt.Errorf("failed to count users: %w", err)
    }
    return counts.TotalAll, counts.TotalActive, counts.TotalNew, counts.TotalAll - counts.TotalActive, nil
}

func (repo *UserRepository) GetUserActivity(offset, limit int) ([]model.UserActivity, error) {
    var userActivities []model.UserActivity
    
    var userIDs []uuid.UUID
    if err := repo.DB.Model(&model.UserActivity{}).
        Select("DISTINCT user_id").
        Order("user_id").
        Offset(offset).Limit(limit).
        Find(&userIDs).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch distinct user ids: %v", err)
    }

    if len(userIDs) == 0 {
        return []model.UserActivity{}, nil
    }
    if err := repo.DB.
        Preload("User").
        Preload("User.UserProfile").
        Preload("User.UserProfile.School").
        Preload("User.UserProfile.Grade").
        Preload("User.UserProfile.Eccd").
        Preload("User.UserProfile.Dzongkhag").
        Where("user_id IN (?)", userIDs).
        Find(&userActivities).Error; err != nil {
        return nil, fmt.Errorf("failed to fetch user activities: %v", err)
    }
    
    return userActivities, nil
}


// applyUserSearch narrows a users query by one field; usersTable is the table or alias holding
// dcdd_auth.dcdd_users. searchBy is matched against a fixed list so it can never reach the SQL as
// a column name. Name and email match any part of the value ignoring case; cid and phone must
// match exactly.
func applyUserSearch(db *gorm.DB, usersTable, searchBy, search string) (*gorm.DB, error) {
    search = strings.TrimSpace(search)
    if search == "" {
        return db, nil
    }

    switch searchBy {
    case "name":
        return db.Where(usersTable+".id IN (?)", profileUserIDs(db).Where("name ILIKE ?", containsPattern(search))), nil
    case "cid":
        return db.Where(usersTable+".id IN (?)", profileUserIDs(db).Where("cid = ?", search)), nil
    case "email":
        return db.Where(usersTable+".email ILIKE ?", containsPattern(search)), nil
    case "mobile_no":
        return db.Where(usersTable+".mobile_no = ?", search), nil
    default:
        return nil, fmt.Errorf("invalid search_by %q: must be one of name, cid, mobile_no, email", searchBy)
    }
}

func profileUserIDs(db *gorm.DB) *gorm.DB {
    return db.Session(&gorm.Session{NewDB: true}).Model(&model.UserProfile{}).Select("user_id")
}

// containsPattern escapes LIKE wildcards so the value is matched literally.
func containsPattern(search string) string {
    escaped := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(search)
    return "%" + escaped + "%"
}
