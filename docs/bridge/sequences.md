## Deposit

Bridge ERC 20 tokens from rootchain to childchain via deposit.

```mermaid
sequenceDiagram
	User->>Chain: deposit
	Chain->>RootERC20.sol: approve(RootERC20Predicate)
	Chain->>RootERC20Predicate.sol: deposit()
	RootERC20Predicate.sol->>RootERC20Predicate.sol: mapToken()
	RootERC20Predicate.sol->>StateSender.sol: syncState(MAP_TOKEN_SIG), recv=ChildERC20Predicate
	RootERC20Predicate.sol-->>Chain: TokenMapped Event
	StateSender.sol-->>Chain: StateSynced Event to map tokens on child predicate
	RootERC20Predicate.sol->>StateSender.sol: syncState(DEPOSIT_SIG), recv=ChildERC20Predicate
	StateSender.sol-->>Chain: StateSynced Event to deposit on child chain
	Chain->>User: ok
	Chain->>StateReceiver.sol:commit()
	StateReceiver.sol-->>Chain: NewCommitment Event
	Chain->>StateReceiver.sol:execute()
	StateReceiver.sol->>ChildERC20Predicate.sol:onStateReceive()
	ChildERC20Predicate.sol->>ChildERC20.sol: mint()
	StateReceiver.sol-->>Chain:StateSyncResult Event
```

## Withdraw

Bridge ERC 20 tokens from childchain to rootchain via withdrawal.

```mermaid
sequenceDiagram
	User->>Chain: withdraw
	Chain->>ChildERC20Predicate.sol: withdrawTo()
	ChildERC20Predicate.sol->>ChildERC20: burn()
	ChildERC20Predicate.sol->>L2StateSender.sol: syncState(WITHDRAW_SIG), recv=RootERC20Predicate
	Chain->>User: tx hash
	User->>Chain: get tx receipt
	Chain->>User: exit event id
	ChildERC20Predicate.sol-->>Chain: L2ERC20Withdraw Event
	L2StateSender.sol-->>Chain: StateSynced Event
	Chain->>Chain: Seal block
	Chain->>CheckpointManager.sol: submit()
```
## Exit

Finalize withdrawal of ERC 20 tokens from childchain to rootchain.

```mermaid
sequenceDiagram
	User->>Chain: exit, event id:X
	Chain->>Chain: bridge_generateExitProof()
	Chain->>CheckpointManager.sol: getCheckpointBlock()
	CheckpointManager.sol->>Chain: blockNum
	Chain->>Chain: getExitEventsForProof(epochNum, blockNum)
	Chain->>Chain: createExitTree(exitEvents)
	Chain->>Chain: generateProof()
	Chain->>ExitHelper.sol: exit()
	ExitHelper.sol->>CheckpointManager.sol: getEventMembershipByBlockNumber()
	ExitHelper.sol->>RootERC20Predicate.sol:onL2StateReceive()
	RootERC20Predicate.sol->>RootERC20: transfer()
	Chain->>User: ok
	RootERC20Predicate.sol-->>Chain: ERC20Withdraw Event
	ExitHelper.sol-->>Chain: ExitProcessed Event
```

