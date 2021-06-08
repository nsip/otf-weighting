package weight

import "testing"

func Test_utc2dt(t *testing.T) {
	type args struct {
		utc string
	}
	tests := []struct {
		name   string
		args   args
		wantDt string
		wantTm string
	}{
		{
			name: "OK",
			args: args{
				utc: "2021-06-08T07:05:30Z",
			},
			wantDt: "20210608",
			wantTm: "070530",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDt, gotTm := utc2dtm(tt.args.utc)
			if gotDt != tt.wantDt {
				t.Errorf("utc2dtm() gotDt = %v, want %v", gotDt, tt.wantDt)
			}
			if gotTm != tt.wantTm {
				t.Errorf("utc2dtm() gotTm = %v, want %v", gotTm, tt.wantTm)
			}
		})
	}
}
