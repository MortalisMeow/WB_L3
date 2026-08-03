package service

import (
	"context"
	"encoding/json"
	"strings"

	"warehousecontrol/internal/models"
	"warehousecontrol/internal/repo"
)

type Service struct {
	repo *repo.Repo
}

func New(r *repo.Repo) *Service {
	return &Service{repo: r}
}

var users = map[string]string{
	"admin":   models.RoleAdmin,
	"manager": models.RoleManager,
	"viewer":  models.RoleViewer,
}

func (s *Service) Login(in models.LoginInput) (string, string, string, error) {
	username := strings.TrimSpace(in.Username)
	role := strings.TrimSpace(in.Role)
	if username == "" {
		username = role
	}
	if role == "" {
		if r, ok := users[username]; ok {
			role = r
		}
	}
	if role != models.RoleAdmin && role != models.RoleManager && role != models.RoleViewer {
		return "", "", "", models.ErrInvalidRole
	}
	return username, role, role, nil
}

func (s *Service) Create(ctx context.Context, username string, in models.CreateInput) (*models.Item, error) {
	item, err := s.validate(in.Name, in.SKU, in.Quantity, in.Price, in.Description)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, username, item)
}

func (s *Service) Get(ctx context.Context, id int64) (*models.Item, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]models.Item, error) {
	return s.repo.List(ctx)
}

func (s *Service) Update(ctx context.Context, username string, id int64, in models.UpdateInput) (*models.Item, error) {
	item, err := s.validate(in.Name, in.SKU, in.Quantity, in.Price, in.Description)
	if err != nil {
		return nil, err
	}
	item.ID = id
	if err := s.repo.Update(ctx, username, item); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, username string, id int64) error {
	return s.repo.Delete(ctx, username, id)
}

func (s *Service) ItemHistory(ctx context.Context, itemID int64, f models.HistoryFilter) ([]models.HistoryEntry, error) {
	f.ItemID = itemID
	return s.repo.ListHistory(ctx, f)
}

func (s *Service) History(ctx context.Context, f models.HistoryFilter) ([]models.HistoryEntry, error) {
	return s.repo.ListHistory(ctx, f)
}

func (s *Service) HistoryDiff(ctx context.Context, historyID int64) (*models.HistoryDiff, error) {
	entry, err := s.repo.GetHistoryByID(ctx, historyID)
	if err != nil {
		return nil, err
	}
	return &models.HistoryDiff{
		Entry:   *entry,
		Changes: buildDiff(entry.OldData, entry.NewData),
	}, nil
}

func (s *Service) validate(name, sku string, qty int, price float64, desc string) (*models.Item, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, models.ErrInvalidName
	}
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return nil, models.ErrInvalidSKU
	}
	if qty < 0 {
		return nil, models.ErrInvalidQty
	}
	if price < 0 {
		return nil, models.ErrInvalidPrice
	}
	return &models.Item{
		Name:        name,
		SKU:         sku,
		Quantity:    qty,
		Price:       price,
		Description: strings.TrimSpace(desc),
	}, nil
}

func buildDiff(oldRaw, newRaw json.RawMessage) map[string]models.DiffPair {
	fields := []string{"name", "sku", "quantity", "price", "description"}
	oldMap := parseJSONMap(oldRaw)
	newMap := parseJSONMap(newRaw)
	changes := make(map[string]models.DiffPair)

	for _, f := range fields {
		oldVal, oldOk := oldMap[f]
		newVal, newOk := newMap[f]
		if !oldOk && !newOk {
			continue
		}
		if jsonEqual(oldVal, newVal) {
			continue
		}
		changes[f] = models.DiffPair{Old: oldVal, New: newVal}
	}
	return changes
}

func parseJSONMap(raw json.RawMessage) map[string]interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]interface{}{}
	}
	delete(m, "id")
	delete(m, "created_at")
	delete(m, "updated_at")
	return m
}

func jsonEqual(a, b interface{}) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}
