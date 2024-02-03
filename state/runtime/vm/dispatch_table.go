package evm

import "fmt"

type handler struct {
	inst  instruction
	stack int
	gas   uint64
}

var dispatchTable [256]handler

func register(op OpCode, h handler) {
	if dispatchTable[op].inst != nil {
		panic(fmt.Errorf("instruction already exists")) //nolint:gocritic
	}

	dispatchTable[op] = h
}

func registerRange(from, to OpCode, factory func(n int) instruction, gas uint64) {
	c := 1
	for i := from; i <= to; i++ {
		register(i, handler{factory(c), 0, gas})
		c++
	}
}

func init() {
	// unsigned arithmetic operations
	register(STOP, handler{opStop, 0, 0})
	register(ADD, handler{opAdd, 2, 1})
	register(SUB, handler{opSub, 2, 1})
	register(MUL, handler{opMul, 2, 2})
	register(DIV, handler{opDiv, 2, 2})
	register(SDIV, handler{opSDiv, 2, 2})
	register(MOD, handler{opMod, 2, 2})
	register(SMOD, handler{opSMod, 2, 2})
	register(EXP, handler{opExp, 2, 3})

	registerRange(PUSH1, PUSH32, opPush, 1)
	registerRange(DUP1, DUP16, opDup, 1)
	registerRange(SWAP1, SWAP16, opSwap, 1)
	registerRange(LOG0, LOG4, opLog, 175)

	register(ADDMOD, handler{opAddMod, 3, 3})
	register(MULMOD, handler{opMulMod, 3, 3})

	register(AND, handler{opAnd, 2, 1})
	register(OR, handler{opOr, 2, 1})
	register(XOR, handler{opXor, 2, 1})
	register(BYTE, handler{opByte, 2, 1})

	register(NOT, handler{opNot, 1, 1})
	register(ISZERO, handler{opIsZero, 1, 1})

	register(EQ, handler{opEq, 2, 1})
	register(LT, handler{opLt, 2, 1})
	register(GT, handler{opGt, 2, 1})
	register(SLT, handler{opSlt, 2, 1})
	register(SGT, handler{opSgt, 2, 1})

	register(SIGNEXTEND, handler{opSignExtension, 1, 2})

	register(SHL, handler{opShl, 2, 1})
	register(SHR, handler{opShr, 2, 1})
	register(SAR, handler{opSar, 2, 1})

	register(CREATE, handler{opCreate(CREATE), 3, 11000})
	register(CREATE2, handler{opCreate(CREATE2), 4, 11000})

	register(CALL, handler{opCall(CALL), 7, 0})
	register(CALLCODE, handler{opCall(CALLCODE), 7, 0})
	register(DELEGATECALL, handler{opCall(DELEGATECALL), 6, 0})
	register(STATICCALL, handler{opCall(STATICCALL), 6, 0})

	register(REVERT, handler{opHalt(REVERT), 2, 0})
	register(RETURN, handler{opHalt(RETURN), 2, 0})

	// memory
	register(MLOAD, handler{opMload, 1, 1})
	register(MSTORE, handler{opMStore, 2, 1})
	register(MSTORE8, handler{opMStore8, 2, 1})

	// store
	register(SLOAD, handler{opSload, 1, 0})
	register(SSTORE, handler{opSStore, 2, 0})

	register(SHA3, handler{opSha3, 2, 10})

	register(POP, handler{opPop, 1, 1})

	register(EXTCODEHASH, handler{opExtCodeHash, 1, 0})

	// context operations
	register(ADDRESS, handler{opAddress, 0, 0})
	register(BALANCE, handler{opBalance, 1, 0})
	register(SELFBALANCE, handler{opSelfBalance, 0, 1})
	register(ORIGIN, handler{opOrigin, 0, 1})
	register(CALLER, handler{opCaller, 0, 1})
	register(CALLVALUE, handler{opCallValue, 0, 1})
	register(CALLDATALOAD, handler{opCallDataLoad, 1, 1})
	register(CALLDATASIZE, handler{opCallDataSize, 0, 1})
	register(CODESIZE, handler{opCodeSize, 0, 1})
	register(EXTCODESIZE, handler{opExtCodeSize, 1, 0})
	register(GASPRICE, handler{opGasPrice, 0, 1})
	register(RETURNDATASIZE, handler{opReturnDataSize, 0, 1})
	register(CHAINID, handler{opChainID, 0, 1})
	register(PC, handler{opPC, 0, 1})
	register(MSIZE, handler{opMSize, 0, 1})
	register(GAS, handler{opGas, 0, 1})

	register(EXTCODECOPY, handler{opExtCodeCopy, 4, 0})

	register(CALLDATACOPY, handler{opCallDataCopy, 3, 1})
	register(RETURNDATACOPY, handler{opReturnDataCopy, 3, 1})
	register(CODECOPY, handler{opCodeCopy, 3, 1})

	// block information
	register(BLOCKHASH, handler{opBlockHash, 1, 5})
	register(COINBASE, handler{opCoinbase, 0, 1})
	register(TIMESTAMP, handler{opTimestamp, 0, 1})
	register(NUMBER, handler{opNumber, 0, 1})
	register(DIFFICULTY, handler{opDifficulty, 0, 1})
	register(GASLIMIT, handler{opGasLimit, 0, 1})
	register(BASEFEE, handler{opBaseFee, 0, 1})

	register(SELFDESTRUCT, handler{opSelfDestruct, 1, 0})

	// jumps
	register(JUMP, handler{opJump, 1, 3})
	register(JUMPI, handler{opJumpi, 2, 5})
	register(JUMPDEST, handler{opJumpDest, 0, 1})
}
