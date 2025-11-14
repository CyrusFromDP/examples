require("@nomicfoundation/hardhat-toolbox");

/** @type import('hardhat/config').HardhatUserConfig */
module.exports = {
  solidity: "0.8.28",
  networks: {
    pharos: {
      url: "https://atlantic.dplabs-internal.com",
      accounts: [vars.get("PRIVATE_KEY")],
    },
  },
  etherscan: {
    customChains: [
      {
        network: "pharos",
        chainId: 688689,
        urls: {
          apiURL: "https://api.socialscan.io/pharos-atlantic-testnet/v1/explorer/command_api/contract",
          browserURL: "https://pharos-atlantic-testnet.socialscan.io",
        },
      },
    ],
    apiKey: {
      pharos: "Put a random string", // Note we don't need a apiKey here, just leave a random string
    },
  }
};