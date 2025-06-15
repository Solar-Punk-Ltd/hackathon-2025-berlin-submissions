import { ethers } from "hardhat";
import { DataContract } from "../typechain-types";

async function main() {
  console.log("=== DataContract Interaction Script (Sepolia) ===\n");

  // Get the deployed contract address
  const contractAddress = process.env.CONTRACT_ADDRESS;
  if (!contractAddress) {
    console.error("Error: CONTRACT_ADDRESS environment variable not set");
    console.log("Please set CONTRACT_ADDRESS to the deployed contract address");
    console.log("Example: CONTRACT_ADDRESS=0x... npm run interact");
    process.exit(1);
  }

  // Get signers
  const signers = await ethers.getSigners();
  if (signers.length === 0) {
    console.error("Error: No signers available");
    process.exit(1);
  }

  const deployer = signers[0];
  console.log("Deployer address:", deployer.address);
  
  // For single signer scenarios, we'll use the same address as target
  const targetAddress = deployer.address;

  // Connect to the deployed contract
  const DataContract = await ethers.getContractFactory("DataContract");
  const dataContract = DataContract.attach(contractAddress) as DataContract;

  console.log("\n=== Contract Information ===");
  console.log("Contract address:", contractAddress);
  console.log("Network:", await ethers.provider.getNetwork());
  console.log("Current caller:", deployer.address);

  console.log("\n=== Sending Data to Target ===");
  
  // Example data to send
  const ownerParam = ethers.encodeBytes32String("OWNER_001");
  const actref = ethers.encodeBytes32String("ACTION_REF_123");
  const topic = "Test Topic - Sepolia Contract Interaction";

  console.log("Target address:", targetAddress);
  console.log("Owner param:", ownerParam);
  console.log("Action reference:", actref);
  console.log("Topic:", topic);

  try {
    // Check balance first
    const balance = await ethers.provider.getBalance(deployer.address);
    console.log("Account balance:", ethers.formatEther(balance), "ETH");

    // Send data to target
    console.log("\nSending data to target...");
    const tx = await dataContract.sendDataToTarget(
      targetAddress,
      ownerParam,
      actref,
      topic
    );
    
    console.log("Transaction hash:", tx.hash);
    console.log("Waiting for confirmation...");
    
    const receipt = await tx.wait();
    if (!receipt) {
      console.error("Transaction receipt is null");
      return;
    }
    
    console.log("Transaction confirmed in block:", receipt.blockNumber);
    console.log("Gas used:", receipt.gasUsed.toString());

    // Check the event logs
    console.log(`Found ${receipt.logs.length} logs in transaction`);
    for (let i = 0; i < receipt.logs.length; i++) {
      const log = receipt.logs[i];
      try {
        const parsedLog = dataContract.interface.parseLog({
          topics: log.topics,
          data: log.data
        });
        if (parsedLog && parsedLog.name === 'DataSentToTarget') {
          console.log("\n=== Event Details ===");
          console.log("From:", parsedLog.args[0]);
          console.log("To:", parsedLog.args[1]);
          console.log("Owner:", parsedLog.args[2]);
          console.log("Action Ref:", parsedLog.args[3]);
          console.log("Topic:", parsedLog.args[4]);
        }
      } catch (error) {
        // Log might not be from our contract, ignore
      }
    }

  } catch (error: any) {
    console.error("Error sending data:", error.message);
    if (error.reason) {
      console.error("Reason:", error.reason);
    }
  }

  console.log("\n=== Multiple Data Sends ===");
  
  // Send multiple data entries
  const dataEntries = [
    {
      target: targetAddress,
      owner: ethers.encodeBytes32String("OWNER_002"),
      actref: ethers.encodeBytes32String("REF_BATCH_001"),
      topic: "Sepolia Batch Entry 1"
    },
    {
      target: targetAddress,
      owner: ethers.encodeBytes32String("OWNER_003"),
      actref: ethers.encodeBytes32String("REF_BATCH_002"),
      topic: "Sepolia Batch Entry 2"
    }
  ];

  for (let i = 0; i < dataEntries.length; i++) {
    const entry = dataEntries[i];
    console.log(`\nSending batch entry ${i + 1}/${dataEntries.length}...`);
    
    try {
      const tx = await dataContract.sendDataToTarget(
        entry.target,
        entry.owner,
        entry.actref,
        entry.topic
      );
      
      console.log("Transaction hash:", tx.hash);
      const receipt = await tx.wait();
      if (receipt) {
        console.log(`✓ Batch entry ${i + 1} sent successfully (Block: ${receipt.blockNumber})`);
      } else {
        console.log(`✗ Batch entry ${i + 1} failed: Receipt is null`);
      }
      
    } catch (error: any) {
      console.error(`✗ Batch entry ${i + 1} failed:`, error.message);
    }
  }

  console.log("\n=== Interaction Complete ===");
  console.log("Contract interaction script finished successfully!");
  console.log("View transactions on Sepolia Etherscan:");
  console.log(`https://sepolia.etherscan.io/address/${contractAddress}`);
}

// Execute the script
main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("Script failed:", error);
    process.exit(1);
  });
