package gosii

import (
	"testing"
)

func TestConsulta_GetNameByRUT(t *testing.T) {
	ssiClient := NewClient(nil)
	data, err := ssiClient.GetNameByRUT("5.126.663-3")
	checkResultOk(t, err, data)

	data, err = ssiClient.GetNameByRUT("5126.6633")
	checkResultOk(t, err, data)

	data, err = ssiClient.GetNameByRUT("51266633")
	checkResultOk(t, err, data)

	_, err = ssiClient.GetNameByRUT("51266633111")
	if err == nil {
		t.Errorf("GetNameByRUT() error = %v", err)
	}
}

func checkResultOk(t *testing.T, err error, data *Citizen) {
	if err != nil {
		t.Errorf("GetNameByRUT() error = %v", err)
	}
	if data.Name != "MIGUEL JUAN SEBASTIAN PINERA ECHENIQUE" {
		t.Errorf("GetNameByRUT() got = %v, want %v", data.Name, "MIGUEL JUAN SEBASTIAN PINERA ECHENIQUE")
	}
}
