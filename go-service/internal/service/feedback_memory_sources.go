package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"github.com/google/uuid"
)

// normalizeFeedbackMemoryVersion 将旧版previous引用展开为稳定的原始素材引用，跨代仍可撤销和追溯。
func normalizeFeedbackMemoryVersion(v *model.FeedbackMemoryVersion, load func(uuid.UUID) (*model.FeedbackMemoryVersion, error), depth int) ([]model.FeedbackMemoryItem, []FeedbackMemorySource, error) {
	if depth > 256 {
		return nil, nil, fmt.Errorf("记忆来源链过深")
	}
	var items []model.FeedbackMemoryItem
	var sources []FeedbackMemorySource
	if err := json.Unmarshal(v.Items, &items); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(v.Sources, &sources); err != nil {
		return nil, nil, err
	}
	originals := map[string]FeedbackMemorySource{}
	legacy := map[string]bool{}
	for _, source := range sources {
		if source.SourceType == "previous" {
			legacy[source.Ref] = true
		} else {
			originals[source.Ref] = source
		}
	}
	needsParent := false
	for _, item := range items {
		for _, ref := range item.SourceRefs {
			if legacy[ref] {
				needsParent = true
			}
		}
	}
	var parentItems []model.FeedbackMemoryItem
	if needsParent {
		if v.BaseVersionID == nil {
			return nil, nil, fmt.Errorf("旧记忆缺少来源基线")
		}
		parent, err := load(*v.BaseVersionID)
		if err != nil {
			return nil, nil, err
		}
		if parent.MemoryID != v.MemoryID || parent.TenantID != v.TenantID {
			return nil, nil, fmt.Errorf("记忆来源越界")
		}
		var parentSources []FeedbackMemorySource
		parentItems, parentSources, err = normalizeFeedbackMemoryVersion(parent, load, depth+1)
		if err != nil {
			return nil, nil, err
		}
		for _, source := range parentSources {
			originals[source.Ref] = source
		}
	}
	outSources := []FeedbackMemorySource{}
	used := map[string]bool{}
	for i := range items {
		refs := []string{}
		seen := map[string]bool{}
		for _, ref := range items[i].SourceRefs {
			expanded := []string{ref}
			if legacy[ref] {
				parts := strings.Split(ref, ":")
				index, err := strconv.Atoi(parts[len(parts)-1])
				if err != nil || index < 1 || index > len(parentItems) {
					return nil, nil, fmt.Errorf("旧记忆来源索引无效")
				}
				expanded = parentItems[index-1].SourceRefs
			}
			for _, original := range expanded {
				source, ok := originals[original]
				if !ok {
					return nil, nil, fmt.Errorf("记忆原始来源缺失")
				}
				if !seen[original] {
					refs = append(refs, original)
					seen[original] = true
				}
				if !used[original] {
					outSources = append(outSources, source)
					used[original] = true
				}
			}
		}
		items[i].SourceRefs = refs
	}
	return items, outSources, nil
}

// buildFeedbackMemorySources 只送入新素材及旧记忆实际引用的依据；删除标记永不成为正向来源。
func buildFeedbackMemorySources(w *dto.FeedbackMemoryWorkspace, items []model.FeedbackMemoryItem, inherited []FeedbackMemorySource) ([]FeedbackMemorySource, []model.FeedbackMemoryItem, []FeedbackMemorySource) {
	sources := []FeedbackMemorySource{}
	previous := []model.FeedbackMemoryItem{}
	deleted := []FeedbackMemorySource{}
	removed := map[string]bool{}
	oldRefs := map[string]bool{}
	for _, source := range inherited {
		oldRefs[source.Ref] = true
	}
	for _, entry := range w.DeletedEntries {
		ref := entry.ID.String()
		removed[ref] = true
		if oldRefs[ref] || w.Baseline == nil || (entry.DeletedAt != nil && !entry.DeletedAt.Before(w.Baseline.CreatedAt)) {
			deleted = append(deleted, FeedbackMemorySource{Ref: ref, SourceType: entry.SourceType, Condition: entry.Condition, Content: entry.Content, Snapshot: entry})
		}
	}
	needed := map[string]bool{}
	for _, item := range items {
		refs := []string{}
		for _, ref := range item.SourceRefs {
			if !removed[ref] {
				refs = append(refs, ref)
				needed[ref] = true
			}
		}
		if len(refs) > 0 {
			item.SourceRefs = refs
			previous = append(previous, item)
		}
	}
	for _, source := range inherited {
		if needed[source.Ref] && !removed[source.Ref] {
			source.Inherited = true
			sources = append(sources, source)
		}
	}
	for _, entry := range w.Entries {
		if entry.Enabled && entry.ImplantedVersionID == nil && !needed[entry.ID.String()] {
			sources = append(sources, FeedbackMemorySource{Ref: entry.ID.String(), SourceType: entry.SourceType, Condition: entry.Condition, Content: entry.Content, Snapshot: entry})
		}
	}
	return sources, previous, deleted
}

// mergeFeedbackMemoryDuplicates 合并完全相同的条目并保留所有来源；语义合并由AI按适用范围处理。
func mergeFeedbackMemoryDuplicates(items []model.FeedbackMemoryItem) []model.FeedbackMemoryItem {
	out := []model.FeedbackMemoryItem{}
	indices := map[string]int{}
	for _, item := range items {
		item.Content = strings.TrimSpace(item.Content)
		item.Condition = strings.TrimSpace(item.Condition)
		key := item.Condition + "\x00" + item.Content
		if i, ok := indices[key]; ok {
			seen := map[string]bool{}
			for _, ref := range out[i].SourceRefs {
				seen[ref] = true
			}
			for _, ref := range item.SourceRefs {
				if !seen[ref] {
					out[i].SourceRefs = append(out[i].SourceRefs, ref)
					seen[ref] = true
				}
			}
		} else {
			indices[key] = len(out)
			out = append(out, item)
		}
	}
	return out
}
