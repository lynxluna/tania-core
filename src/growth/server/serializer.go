package server

import (
	"net/http"
	"sort"
	"time"

	"github.com/Tanibox/tania-server/src/growth/query"

	"github.com/Tanibox/tania-server/src/growth/domain"
	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
)

type CropBatch struct {
	UID              uuid.UUID                      `json:"uid"`
	BatchID          string                         `json:"batch_id"`
	Status           string                         `json:"status"`
	Type             string                         `json:"type"`
	Container        CropContainer                  `json:"container"`
	Inventory        query.CropInventoryQueryResult `json:"inventory"`
	CreatedDate      time.Time                      `json:"created_date"`
	DaysSinceSeeding int                            `json:"days_since_seeding"`
	Photo            domain.CropPhoto               `json:"photo"`
	ActivityType     CropActivityType               `json:"activity_type"`

	InitialArea InitialArea `json:"initial_area"`
	MovedArea   []MovedArea `json:"moved_area"`

	// Fields to track care crop
	LastFertilized string `json:"last_fertilized"`
	LastPruned     string `json:"last_pruned"`
	LastPesticided string `json:"last_pesticided"`

	Notes SortedCropNotes `json:"notes"`
}

type CropContainer struct {
	Quantity int               `json:"quantity"`
	Type     CropContainerType `json:"type"`
}
type CropContainerType struct {
	Code string `json:"code"`
	Cell int    `json:"cell"`
}

type InitialArea struct {
	Area            query.CropAreaQueryResult `json:"area"`
	InitialQuantity int                       `json:"initial_quantity"`
	CurrentQuantity int                       `json:"current_quantity"`
}

type MovedArea struct {
	Area            query.CropAreaQueryResult `json:"area"`
	SourceArea      query.CropAreaQueryResult `json:"source_area"`
	InitialQuantity int                       `json:"initial_quantity"`
	CurrentQuantity int                       `json:"current_quantity"`
	CreatedDate     time.Time                 `json:"created_date"`
	LastUpdated     time.Time                 `json:"last_updated"`
}

type SortedCropNotes []domain.CropNote

// Len is part of sort.Interface.
func (sn SortedCropNotes) Len() int { return len(sn) }

// Swap is part of sort.Interface.
func (sn SortedCropNotes) Swap(i, j int) { sn[i], sn[j] = sn[j], sn[i] }

// Less is part of sort.Interface.
func (sn SortedCropNotes) Less(i, j int) bool { return sn[i].CreatedDate.After(sn[j].CreatedDate) }

type CropActivityType struct {
	TotalSeeding int `json:"total_seeding"`
	TotalGrowing int `json:"total_growing"`
}

func MapToCropBatch(s *GrowthServer, crop domain.Crop) (CropBatch, error) {
	queryResult := <-s.InventoryMaterialQuery.FindByID(crop.InventoryUID)
	if queryResult.Error != nil {
		return CropBatch{}, queryResult.Error
	}

	cropInventory, ok := queryResult.Result.(query.CropInventoryQueryResult)
	if !ok {
		return CropBatch{}, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
	}

	totalSeeding := 0
	totalGrowing := 0

	queryResult = <-s.AreaQuery.FindByID(crop.InitialArea.AreaUID)
	if queryResult.Error != nil {
		return CropBatch{}, queryResult.Error
	}

	initialArea, ok := queryResult.Result.(query.CropAreaQueryResult)
	if !ok {
		return CropBatch{}, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
	}

	if initialArea.Type == "SEEDING" {
		totalSeeding += crop.InitialArea.CurrentQuantity
	} else if initialArea.Type == "GROWING" {
		totalGrowing += crop.InitialArea.CurrentQuantity
	}

	movedAreas := []MovedArea{}
	for _, v := range crop.MovedArea {
		queryResult = <-s.AreaQuery.FindByID(v.AreaUID)
		if queryResult.Error != nil {
			return CropBatch{}, queryResult.Error
		}

		area, ok := queryResult.Result.(query.CropAreaQueryResult)
		if !ok {
			return CropBatch{}, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
		}

		queryResult = <-s.AreaQuery.FindByID(v.SourceAreaUID)
		if queryResult.Error != nil {
			return CropBatch{}, queryResult.Error
		}

		sourceArea, ok := queryResult.Result.(query.CropAreaQueryResult)
		if !ok {
			return CropBatch{}, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
		}

		if area.Type == "SEEDING" {
			totalSeeding += v.CurrentQuantity
		}
		if area.Type == "GROWING" {
			totalGrowing += v.CurrentQuantity
		}

		movedAreas = append(movedAreas, MovedArea{
			Area:            area,
			SourceArea:      sourceArea,
			InitialQuantity: v.InitialQuantity,
			CurrentQuantity: v.CurrentQuantity,
			CreatedDate:     v.CreatedDate,
			LastUpdated:     v.LastUpdated,
		})
	}

	cropBatch := CropBatch{}
	cropBatch.UID = crop.UID
	cropBatch.BatchID = crop.BatchID
	cropBatch.Status = crop.Status.Code
	cropBatch.Type = crop.Type.Code

	code := ""
	cell := 0
	switch v := crop.Container.Type.(type) {
	case domain.Tray:
		code = v.Code()
		cell = v.Cell
	case domain.Pot:
		code = v.Code()
	}
	cropBatch.Container = CropContainer{
		Quantity: crop.Container.Quantity,
		Type: CropContainerType{
			Code: code,
			Cell: cell,
		},
	}

	cropBatch.Inventory = query.CropInventoryQueryResult{
		UID:           cropInventory.UID,
		PlantTypeCode: cropInventory.PlantTypeCode,
		Variety:       cropInventory.Variety,
	}
	cropBatch.CreatedDate = crop.CreatedDate
	cropBatch.DaysSinceSeeding = crop.CalculateDaysSinceSeeding()

	cropBatch.InitialArea = InitialArea{
		Area:            initialArea,
		InitialQuantity: crop.InitialArea.InitialQuantity,
		CurrentQuantity: crop.InitialArea.CurrentQuantity,
	}
	cropBatch.MovedArea = movedAreas

	cropBatch.LastFertilized = ""
	if !crop.LastFertilized.IsZero() {
		cropBatch.LastFertilized = crop.LastFertilized.String()
	}
	cropBatch.LastPesticided = ""
	if !crop.LastPesticided.IsZero() {
		cropBatch.LastPesticided = crop.LastPesticided.String()
	}
	cropBatch.LastPruned = ""
	if !crop.LastPruned.IsZero() {
		cropBatch.LastPruned = crop.LastPruned.String()
	}

	cropBatch.ActivityType = CropActivityType{
		TotalSeeding: totalSeeding,
		TotalGrowing: totalGrowing,
	}

	notes := make(SortedCropNotes, 0, len(crop.Notes))
	for _, v := range crop.Notes {
		notes = append(notes, v)
	}

	sort.Sort(notes)

	cropBatch.Notes = notes

	cropBatch.Photo = crop.Photo

	return cropBatch, nil
}
