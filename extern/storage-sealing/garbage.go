package sealing

import (
	"context"
	"fmt"
	"github.com/filecoin-project/lotus/extern/sector-storage/storiface"
	"github.com/mitchellh/go-homedir"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/specs-storage/storage"
)

func (m *Sealing) PledgeSector(ctx context.Context) (storage.SectorRef, error) {
	log.Infof("第三层", ctx)
	m.startupWait.Wait()

	m.inputLk.Lock()
	defer m.inputLk.Unlock()

	cfg, err := m.getConfig()
	if err != nil {
		return storage.SectorRef{}, xerrors.Errorf("getting config: %w", err)
	}

	if cfg.MaxSealingSectors > 0 {
		if m.stats.curSealing() >= cfg.MaxSealingSectors {
			return storage.SectorRef{}, xerrors.Errorf("too many sectors sealing (curSealing: %d, max: %d)", m.stats.curSealing(), cfg.MaxSealingSectors)
		}
	}

	spt, err := m.currentSealProof(ctx)
	if err != nil {
		return storage.SectorRef{}, xerrors.Errorf("getting seal proof type: %w", err)
	}

	sid, err := m.createSector(ctx, cfg, spt)
	if err != nil {
		return storage.SectorRef{}, err
	}

	// added by jack
	//workerid := string(ctx.Value("workerid").([]byte))
	log.Infof("传到里面的ctx", ctx)
	workerid := ""
	if ctx.Value("") != nil {
		workerid = ctx.Value("").(string)
	} else {
		log.Infof("still valid")
	}
	log.Infof("------------------probe used as detect preallocated task to workerid!, sid=%d, workerid=%s", sid, workerid)
	if workerid != "" {
		if homedir, err := homedir.Expand("~"); err == nil {
			for i := 0; i < 2; i++ {
				_, err := os.Stat(filepath.Join(homedir, "./FixedSectorWorkerId"))
				notexist := os.IsNotExist(err)
				if notexist {
					err = os.MkdirAll(filepath.Join(homedir, "./FixedSectorWorkerId"), 0755)
					if err == nil {
						break
					}
				} else {
					id := m.minerSectorID(sid)
					err := os.WriteFile(path.Join(homedir, "./FixedSectorWorkerId", storiface.SectorName(id)+".cfg"), []byte(workerid), 0666)
					if err == nil {
						fmt.Println("Created CC sectgor: ", storiface.SectorName(id), " Pre-allocated to worker: ", workerid)
						break
					}
				}
			}
		}
	}
	//ending

	log.Infof("Creating CC sector %d", sid)
	return m.minerSector(spt, sid), m.sectors.Send(uint64(sid), SectorStartCC{
		ID:         sid,
		SectorType: spt,
	})
}
