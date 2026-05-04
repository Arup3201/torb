package fixtures

import (
	"fmt"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
	"github.com/google/uuid"
)

func RandomProjectRow(ownerID string) models.Project {
	pId := uuid.NewString()

	desc := "Description " + pId
	skills := "C++, Python"

	return models.Project{
		ID:          pId,
		Name:        "Test Project" + pId,
		Description: &desc,
		Skills:      &skills,
		OwnerID:     ownerID,
	}
}

func (f *Fixtures) InsertProject(p models.Project) string {
	if f.db != nil {
		if err := f.db.WithContext(f.ctx).Create(&p).Error; err != nil {
			panic(fmt.Sprintf("insert project fixture failed: %v", err))
		}
		f.InsertMember(GetMemberRow(p.ID, p.OwnerID, core.ROLE_OWNER))
		return p.ID
	}
	return ""
}
