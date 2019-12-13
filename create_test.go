package gorm_test

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jinzhu/now"
)

func TestCreate(t *testing.T) {
	float := 35.03554004971999
	now := time.Now()
	user := User{Name: "CreateUser", Age: 18, Birthday: &now, UserNum: Num(111), PasswordHash: []byte{'f', 'a', 'k', '4'}, Latitude: float}

	if !DB.NewRecord(user) || !DB.NewRecord(&user) {
		t.Error("User should be new record before create")
	}

	if count := DB.Save(&user).RowsAffected; count != 1 {
		t.Error("There should be one record be affected when create record")
	}

	if DB.NewRecord(user) || DB.NewRecord(&user) {
		t.Error("User should not new record after save")
	}

	var newUser User
	if err := DB.First(&newUser, user.Id).Error; err != nil {
		t.Errorf("No error should happen, but got %v", err)
	}

	if !reflect.DeepEqual(newUser.PasswordHash, []byte{'f', 'a', 'k', '4'}) {
		t.Errorf("User's PasswordHash should be saved ([]byte)")
	}

	if newUser.Age != 18 {
		t.Errorf("User's Age should be saved (int)")
	}

	if newUser.UserNum != Num(111) {
		t.Errorf("User's UserNum should be saved (custom type), but got %v", newUser.UserNum)
	}

	if newUser.Latitude != float {
		t.Errorf("Float64 should not be changed after save")
	}

	if user.CreatedAt.IsZero() {
		t.Errorf("Should have created_at after create")
	}

	if newUser.CreatedAt.IsZero() {
		t.Errorf("Should have created_at after create")
	}

	DB.Model(user).Update("name", "create_user_new_name")
	DB.First(&user, user.Id)
	if user.CreatedAt.Format(time.RFC3339Nano) != newUser.CreatedAt.Format(time.RFC3339Nano) {
		t.Errorf("CreatedAt should not be changed after update")
	}
}

func TestCreateEmptyStrut(t *testing.T) {
	type EmptyStruct struct {
		ID uint
	}
	DB.AutoMigrate(&EmptyStruct{})

	if err := DB.Create(&EmptyStruct{}).Error; err != nil {
		t.Errorf("No error should happen when creating user, but got %v", err)
	}
}

func TestCreateWithExistingTimestamp(t *testing.T) {
	user := User{Name: "CreateUserExistingTimestamp"}

	timeA := now.MustParse("2016-01-01")
	user.CreatedAt = timeA
	user.UpdatedAt = timeA
	DB.Save(&user)

	if user.CreatedAt.UTC().Format(time.RFC3339) != timeA.UTC().Format(time.RFC3339) {
		t.Errorf("CreatedAt should not be changed")
	}

	if user.UpdatedAt.UTC().Format(time.RFC3339) != timeA.UTC().Format(time.RFC3339) {
		t.Errorf("UpdatedAt should not be changed")
	}

	var newUser User
	DB.First(&newUser, user.Id)

	if newUser.CreatedAt.UTC().Format(time.RFC3339) != timeA.UTC().Format(time.RFC3339) {
		t.Errorf("CreatedAt should not be changed")
	}

	if newUser.UpdatedAt.UTC().Format(time.RFC3339) != timeA.UTC().Format(time.RFC3339) {
		t.Errorf("UpdatedAt should not be changed")
	}
}

func TestCreateWithNowFuncOverride(t *testing.T) {
	user1 := User{Name: "CreateUserTimestampOverride"}

	timeA := now.MustParse("2016-01-01")

	// do DB.New() because we don't want this test to affect other tests
	db1 := DB.New()
	// set the override to use static timeA
	db1.SetNowFuncOverride(func() time.Time {
		return timeA
	})
	// call .New again to check the override is carried over as well during clone
	db1 = db1.New()

	db1.Save(&user1)

	if user1.CreatedAt.UTC().Format(time.RFC3339) != timeA.UTC().Format(time.RFC3339) {
		t.Errorf("CreatedAt be using the nowFuncOverride")
	}
	if user1.UpdatedAt.UTC().Format(time.RFC3339) != timeA.UTC().Format(time.RFC3339) {
		t.Errorf("UpdatedAt be using the nowFuncOverride")
	}

	// now create another user with a fresh DB.Now() that doesn't have the nowFuncOverride set
	// to make sure that setting it only affected the above instance

	user2 := User{Name: "CreateUserTimestampOverrideNoMore"}

	db2 := DB.New()

	db2.Save(&user2)

	if user2.CreatedAt.UTC().Format(time.RFC3339) == timeA.UTC().Format(time.RFC3339) {
		t.Errorf("CreatedAt no longer be using the nowFuncOverride")
	}
	if user2.UpdatedAt.UTC().Format(time.RFC3339) == timeA.UTC().Format(time.RFC3339) {
		t.Errorf("UpdatedAt no longer be using the nowFuncOverride")
	}
}

type AutoIncrementUser struct {
	User
	Sequence uint `gorm:"AUTO_INCREMENT"`
}

func TestCreateWithAutoIncrement(t *testing.T) {
	if dialect := os.Getenv("GORM_DIALECT"); dialect != "postgres" {
		t.Skip("Skipping this because only postgres properly support auto_increment on a non-primary_key column")
	}

	DB.AutoMigrate(&AutoIncrementUser{})

	user1 := AutoIncrementUser{}
	user2 := AutoIncrementUser{}

	DB.Create(&user1)
	DB.Create(&user2)

	if user2.Sequence-user1.Sequence != 1 {
		t.Errorf("Auto increment should apply on Sequence")
	}
}

func TestCreateWithNoGORMPrimayKey(t *testing.T) {
	if dialect := os.Getenv("GORM_DIALECT"); dialect == "mssql" {
		t.Skip("Skipping this because MSSQL will return identity only if the table has an Id column")
	}

	jt := JoinTable{From: 1, To: 2}
	err := DB.Create(&jt).Error
	if err != nil {
		t.Errorf("No error should happen when create a record without a GORM primary key. But in the database this primary key exists and is the union of 2 or more fields\n But got: %s", err)
	}
}

func TestCreateWithNoStdPrimaryKeyAndDefaultValues(t *testing.T) {
	animal := Animal{Name: "Ferdinand"}
	if DB.Save(&animal).Error != nil {
		t.Errorf("No error should happen when create a record without std primary key")
	}

	if animal.Counter == 0 {
		t.Errorf("No std primary key should be filled value after create")
	}

	if animal.Name != "Ferdinand" {
		t.Errorf("Default value should be overrided")
	}

	// Test create with default value not overrided
	an := Animal{From: "nerdz"}

	if DB.Save(&an).Error != nil {
		t.Errorf("No error should happen when create an record without std primary key")
	}

	// We must fetch the value again, to have the default fields updated
	// (We can't do this in the update statements, since sql default can be expressions
	// And be different from the fields' type (eg. a time.Time fields has a default value of "now()"
	DB.Model(Animal{}).Where(&Animal{Counter: an.Counter}).First(&an)

	if an.Name != "galeone" {
		t.Errorf("Default value should fill the field. But got %v", an.Name)
	}
}

func TestAnonymousScanner(t *testing.T) {
	user := User{Name: "anonymous_scanner", Role: Role{Name: "admin"}}
	DB.Save(&user)

	var user2 User
	DB.First(&user2, "name = ?", "anonymous_scanner")
	if user2.Role.Name != "admin" {
		t.Errorf("Should be able to get anonymous scanner")
	}

	if !user2.Role.IsAdmin() {
		t.Errorf("Should be able to get anonymous scanner")
	}
}

func TestAnonymousField(t *testing.T) {
	user := User{Name: "anonymous_field", Company: Company{Name: "company"}}
	DB.Save(&user)

	var user2 User
	DB.First(&user2, "name = ?", "anonymous_field")
	DB.Model(&user2).Related(&user2.Company)
	if user2.Company.Name != "company" {
		t.Errorf("Should be able to get anonymous field")
	}
}

func TestSelectWithCreate(t *testing.T) {
	user := getPreparedUser("select_user", "select_with_create")
	DB.Select("Name", "BillingAddress", "CreditCard", "Company", "Emails").Create(user)

	var queryuser User
	DB.Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company").First(&queryuser, user.Id)

	if queryuser.Name != user.Name || queryuser.Age == user.Age {
		t.Errorf("Should only create users with name column")
	}

	if queryuser.BillingAddressID.Int64 == 0 || queryuser.ShippingAddressId != 0 ||
		queryuser.CreditCard.ID == 0 || len(queryuser.Emails) == 0 {
		t.Errorf("Should only create selected relationships")
	}
}

func TestOmitWithCreate(t *testing.T) {
	user := getPreparedUser("omit_user", "omit_with_create")
	DB.Omit("Name", "BillingAddress", "CreditCard", "Company", "Emails").Create(user)

	var queryuser User
	DB.Preload("BillingAddress").Preload("ShippingAddress").
		Preload("CreditCard").Preload("Emails").Preload("Company").First(&queryuser, user.Id)

	if queryuser.Name == user.Name || queryuser.Age != user.Age {
		t.Errorf("Should only create users with age column")
	}

	if queryuser.BillingAddressID.Int64 != 0 || queryuser.ShippingAddressId == 0 ||
		queryuser.CreditCard.ID != 0 || len(queryuser.Emails) != 0 {
		t.Errorf("Should not create omitted relationships")
	}
}

func TestCreateIgnore(t *testing.T) {
	float := 35.03554004971999
	now := time.Now()
	user := User{Name: "CreateUser", Age: 18, Birthday: &now, UserNum: Num(111), PasswordHash: []byte{'f', 'a', 'k', '4'}, Latitude: float}

	if !DB.NewRecord(user) || !DB.NewRecord(&user) {
		t.Error("User should be new record before create")
	}

	if count := DB.Create(&user).RowsAffected; count != 1 {
		t.Error("There should be one record be affected when create record")
	}
	if DB.Dialect().GetName() == "mysql" && DB.Set("gorm:insert_modifier", "IGNORE").Create(&user).Error != nil {
		t.Error("Should ignore duplicate user insert by insert modifier:IGNORE ")
	}
}

func TestCreateOnconflict(t *testing.T) {
	DB.AutoMigrate(&EmailWithIdx{})
	defer func() { DB.DropTableIfExists(&EmailWithIdx{}) }()
	now := time.Now().Round(time.Second)
	email := EmailWithIdx{RegisteredAt: &now}

	// ignore
	if !DB.NewRecord(email) || !DB.NewRecord(&email) {
		t.Error("Email should be new record before create")
	}
	if res := DB.Where(&email).First(&EmailWithIdx{}); !res.RecordNotFound() {
		t.Error(res.Error, "OR No record should been found")
	}
	if res := DB.CreateOnConflict(&email, "IGNORE"); res.RowsAffected != 1 || res.Error != nil {
		if res.Error != nil {
			t.Error(res.Error)
		}
		if res.RowsAffected != 1 {
			t.Error("There should be one record be affected when create record")
		}
	}
	if res := DB.CreateOnConflict(&email, "IGNORE"); res.RowsAffected != 0 || res.Error != nil {
		if DB.Dialect().GetName() != "mssql" && DB.Dialect().GetName() != "postgres" && res.Error != nil {
			t.Error(res.Error)
		}
		if res.RowsAffected != 0 {
			t.Error("There should be zero record be affected when create record")
		}
	}

	// update
	if DB.Dialect().GetName() == "postgres" {
		emailUpdated := EmailWithIdx{UserId: 10086}
		if res := DB.CreateOnConflict(&email, "email_with_idxes_pkey", &emailUpdated); res.RowsAffected != 1 || res.Error != nil {
			t.Error(res.Error, "OR There should be one record be affected when create record")
		}
		out := EmailWithIdx{}
		if res := DB.First(&out, &EmailWithIdx{UserId: 10086}); res.Error != nil {
			t.Error(res.Error)
		}
		if !out.RegisteredAt.Equal(now) || out.UserId != 10086 {
			t.Error(out.UserId, "\n", out.RegisteredAt, "\n", now)
		}
	} else if DB.Dialect().GetName() == "mysql" {
		emailUpdated := EmailWithIdx{UserId: 10000}
		if res := DB.CreateOnConflict(&email, &emailUpdated); res.RowsAffected != 2 || res.Error != nil {
			t.Error(res.Error, "OR RowsAffected should be 2 (by mysql doc)")
		}
		out := EmailWithIdx{}
		if res := DB.First(&out, &EmailWithIdx{UserId: 10000}); res.Error != nil {
			t.Error(res.Error)
		}
		if !out.RegisteredAt.Equal(now) || out.UserId != 10000 {
			t.Error(out.UserId, "\n", out.RegisteredAt, "\n", now)
		}
	}
}

func TestGetOrCreate(t *testing.T) {
	DB.AutoMigrate(&EmailWithIdx{})
	defer func() { DB.DropTableIfExists(&EmailWithIdx{}) }()
	now := time.Now().Round(time.Second)
	email := EmailWithIdx{RegisteredAt: &now}
	out := EmailWithIdx{}

	// GetOrCreate: create
	if res := DB.Where(&email).Attrs(&EmailWithIdx{UserId: 10086}).GetOrCreate(&out); res.RowsAffected != 1 || res.Error != nil {
		t.Error(res.Error, "OR RowsAffected should be 1")
	}
	if out.UserId != 10086 || !out.RegisteredAt.Equal(now) {
		t.Error("Found wrong item")
	}
	createOutId := out.Id
	// Find
	out = EmailWithIdx{}
	if res := DB.Where(&email).Find(&out); res.RowsAffected != 1 || res.Error != nil {
		t.Error(res.Error, "OR RowsAffected should be 1")
		t.Error(out)
	}
	if !out.RegisteredAt.Equal(now) || out.UserId != 10086 || out.Id != createOutId {
		t.Error("Found wrong item")
	}
	// GetOrCreate: get
	out = EmailWithIdx{}
	if res := DB.Where(&email).Attrs(&EmailWithIdx{UserId: 10000}).GetOrCreate(&out); res.RowsAffected != 1 || res.Error != nil {
		t.Error(res.Error, "OR RowsAffected should be 1")
		t.Error(out)
	}
	if !out.RegisteredAt.Equal(now) || out.UserId != 10086 || out.Id != createOutId {
		t.Error("Found wrong item")
	}
	// GetOrCreate: create fail, get again and NotFound
	out = EmailWithIdx{}
	if res := DB.Where(&EmailWithIdx{UserId: 10010}).Attrs(&EmailWithIdx{RegisteredAt: &now}).GetOrCreate(&out); !res.RecordNotFound() {
		t.Error("Expected should be NotFound, but got", res.Error)
	}
}

func TestCreateMany(t *testing.T) {
	DB.Delete(&Email{})
	if res := DB.CreateMany([]interface{}{
		&Email{UserId: 1, Email: "jeff@qq.com"},
		&Email{UserId: 2, Email: "alan@qq.com"},
		&Email{UserId: 3, Email: "alice@qq.com"},
		&Email{UserId: 4, Email: "bob@qq.com"},
	}); res.Error != nil || res.RowsAffected != 4 {
		t.Error(res.Error, "OR RowsAffected should be 4, got %d", res.RowsAffected)
	}
	emails := []Email{}
	DB.Model(&Email{}).Find(&emails)
	if len(emails) != 4 {
		t.Error("count should be 4")
	}
	createAt := time.Time{}
	userIds := [4]int{}
	emailEmails := [4]string{}
	for index, email := range emails {
		if createAt.Equal(time.Time{}) {
			createAt = email.CreatedAt
		} else if !createAt.Equal(email.CreatedAt) {
			t.Errorf("%s, %s", email.CreatedAt, createAt)
		}
		t.Log(email)
		userIds[index] = email.UserId
		emailEmails[index] = email.Email
	}
	if userIds != [4]int{1, 2, 3, 4} {
		t.Errorf("expected []int{1,2,3,4}, but got %d", userIds)
	}
	if emailEmails != [4]string{"jeff@qq.com", "alan@qq.com", "alice@qq.com", "bob@qq.com"} {
		t.Errorf("%v", emailEmails)
	}
	DB.Delete(&Email{})

	// fields not same
	if res := DB.CreateMany([]interface{}{
		&Email{Id: 1, UserId: 1, Email: "jeff@qq.com"},
		&Email{UserId: 2, Email: "alan@qq.com"},
		&Email{UserId: 3, Email: "alice@qq.com"},
		&Email{UserId: 4, Email: "bob@qq.com"},
	}); res.Error.Error() != "createMany objects should have the same fields" {
		t.Error("Expected error createMany objects should have the same fields, got", res.Error.Error())
	}

	// OnConflict IGNORE
	if DB.Dialect().GetName() == "mssql" {
		t.Log("mssqldb driver do not raise error when insert many on conflict")
	} else {
		// Duplicate
		if res := DB.CreateMany([]interface{}{
			&Email{Id: 1, UserId: 1, Email: "jeff@qq.com"},
			&Email{Id: 1, UserId: 2, Email: "alan@qq.com"},
			&Email{Id: 1, UserId: 3, Email: "alice@qq.com"},
			&Email{Id: 1, UserId: 4, Email: "bob@qq.com"},
		}); res.Error == nil {
			t.Error("Expected error Duplicate entry, but got nil")
		}

	}
	if DB.Dialect().GetName() == "mysql" || DB.Dialect().GetName() == "sqlite3" {
		// Ignore
		if res := DB.CreateMany([]interface{}{
			&Email{Id: 1, UserId: 1, Email: "jeff@qq.com"},
			&Email{Id: 1, UserId: 2, Email: "alan@qq.com"},
			&Email{Id: 1, UserId: 3, Email: "alice@qq.com"},
			&Email{Id: 1, UserId: 4, Email: "bob@qq.com"},
		}, "IGNORE"); res.Error != nil {
			t.Error(res.Error)
		}
		emails = []Email{}
		DB.Model(&Email{}).Find(&emails)
		if len(emails) != 1 {
			t.Error("count should be 1, but got", len(emails))
		}
	}

	// OnConflict UPDATE
	DB.Delete(&Email{})
	if DB.Dialect().GetName() == "postgres" || DB.Dialect().GetName() == "mysql" {
		DB.Create(&Email{Id: 1, UserId: 10086, Email: "hello@example.com"})

		var updateOrIgnore []interface{}
		if DB.Dialect().GetName() == "postgres" {
			updateOrIgnore = []interface{}{"emails_pkey", &Email{Id: 100, UserId: 10010}}
		} else {
			updateOrIgnore = []interface{}{&Email{Id: 100, UserId: 10010}}
		}
		if res := DB.CreateMany([]interface{}{
			&Email{Id: 1, UserId: 10086, Email: "hello@example.com"}, // Duplicate
			&Email{Id: 2, UserId: 10086, Email: "hello@example.com"}, // Normal
		}, updateOrIgnore...); res.Error != nil {
			t.Error(res.Error)
		}

		emails := []Email{}
		if DB.Model(&Email{}).Find(&emails); len(emails) != 2 {
			t.Error("Expected 2 emails, got", len(emails))
		}
		emails2 := emails[0]
		emails100 := emails[1]
		if emails2.Id != 2 {
			emails2 = emails[1]
			emails100 = emails[0]
		}
		if emails2.Id != 2 || emails2.UserId != 10086 {
			// insert success
			t.Error("Expected email 2 user ID 10086 , got", emails[0].Id, emails[0].UserId)
		}
		if emails100.Id != 100 || emails100.UserId != 10010 {
			// update success
			t.Error("Expected email 100 user ID 10010 , got", emails[1].Id, emails[1].UserId)
		}
	}
	DB.Delete(&Email{})
}
