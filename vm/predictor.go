package vm

// Num of bits used for addressing in the prediction buffer
const BP_INDEX_BIT_NUM = 5
const BP_INDEX_BITMASK = (1 << BP_INDEX_BIT_NUM) - 1
const BP_BUFFER_SIZE = 1 << BP_INDEX_BIT_NUM

type Branch_Predictor struct {
	PredictionBuffer [BP_BUFFER_SIZE]Bp_Entry
	n_bit            uint8
	_max_counter     uint32
}

type Bp_Entry struct {
	tag     uint32
	counter uint32
	target  uint32
	valid   bool
}

func create_predictor(n_bit uint8) Branch_Predictor {
	return Branch_Predictor{
		n_bit:        n_bit,
		_max_counter: uint32(1<<n_bit) - 1, // uint32(math.Pow(2, float64(n_bit)) - 1),
	}
}

func (bp *Branch_Predictor) Reset(nbit uint8) {
	bp.PredictionBuffer = [BP_BUFFER_SIZE]Bp_Entry{}
	bp.n_bit = nbit
	bp._max_counter = uint32(1<<nbit) - 1
}

// Returns the prediction and target_pc for a branch in given addr.
// true means Taken, false means Not Taken
func (bp *Branch_Predictor) predict(pc uint32) (prediction bool, target uint32) {
	index := pc & BP_INDEX_BITMASK
	tag := pc &^ BP_INDEX_BITMASK

	entry := bp.PredictionBuffer[index]

	// BTB miss. Predict not taken
	if !entry.valid || entry.tag != tag {
		return false, 0
	}

	return entry.counter >= bp._max_counter/2+1, entry.target
}

// This function does not makes a prediction, only returns the target, if there is any entry for the given pc.
// If there is no valid entry, returns 0, false.
func (bp *Branch_Predictor) getTarget(pc uint32) (target uint32, valid bool) {
	index := pc & BP_INDEX_BITMASK
	tag := pc &^ BP_INDEX_BITMASK

	entry := bp.PredictionBuffer[index]

	// BTB miss. Return valid = false
	if !entry.valid || entry.tag != tag {
		return 0, false
	}

	return entry.target, true
}

// Updates the n-bit counter based on the real outcome of the branch.
// returns if the prediction was correct or not.
func (bp *Branch_Predictor) update(pc uint32, target uint32, outcome bool) bool {
	index := pc & BP_INDEX_BITMASK
	tag := pc &^ BP_INDEX_BITMASK

	entry := &bp.PredictionBuffer[index]

	// If tag don't match, reset the counter and tag
	if !entry.valid || entry.tag != tag {
		entry.tag = tag
		entry.counter = 0
		entry.valid = true
	}

	// prediction outcome based on the current counter
	prediction := entry.counter >= bp._max_counter/2+1

	if outcome == true {
		if entry.counter < bp._max_counter {
			entry.counter++
		}
	} else {
		if entry.counter > 0 {
			entry.counter--
		}
	}

	entry.target = target

	return prediction == outcome
}
