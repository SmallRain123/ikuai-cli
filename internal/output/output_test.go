package output

import "testing"

func TestHumanBytes(t *testing.T) {
	cases := map[uint64]string{
		0:              "0 B",
		500:            "500 B",
		2048:           "2.00 KiB",
		5 * 1024 * 1024: "5.00 MiB",
	}
	for in, want := range cases {
		if got := HumanBytes(in); got != want {
			t.Errorf("HumanBytes(%d)=%q want %q", in, got, want)
		}
	}
}

func TestHumanSpeed(t *testing.T) {
	// 500 byte/s = 4000 bps
	if got := HumanSpeed(500); got != "4.00 kbps" {
		t.Errorf("HumanSpeed(500)=%q", got)
	}
	if got := HumanSpeed(1_250_000); got != "10.00 Mbps" { // 10Mbps
		t.Errorf("HumanSpeed(1250000)=%q", got)
	}
}

func TestTable(t *testing.T) {
	// 仅验证不 panic
	Table([]string{"A", "B"}, [][]string{{"1", "2"}, {"3", "4"}})
	Table([]string{"A"}, [][]string{})
}
