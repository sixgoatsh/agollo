package config

import "sort"

type Configurations map[string]interface{}

func (old Configurations) Different(new Configurations) Changes {
	var changes []Change
	for k, newValue := range new {
		if oldValue, ok := old[k]; ok && oldValue != newValue {
			changes = append(changes, NewChange(ChangeTypeUpdate, k, newValue))
		} else {
			changes = append(changes, NewChange(ChangeTypeAdd, k, newValue))
		}
	}

	for k, oldValue := range old {
		_, found := new[k]
		if !found {
			changes = append(changes, NewChange(ChangeTypeDelete, k, oldValue))
		}
	}

	sort.Sort(Changes(changes))

	return changes
}
