// Code generated by "stringer -type=OptNum -trimprefix=Opt"; DO NOT EDIT.

package dhcp

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[OptWordAligned-0]
	_ = x[OptSubnetMask-1]
	_ = x[OptTimeOffset-2]
	_ = x[OptRouter-3]
	_ = x[OptTimeServers-4]
	_ = x[OptNameServers-5]
	_ = x[OptDNSServers-6]
	_ = x[OptLogServers-7]
	_ = x[OptCookieServers-8]
	_ = x[OptLPRServers-9]
	_ = x[OptImpressServers-10]
	_ = x[OptRLPServers-11]
	_ = x[OptHostName-12]
	_ = x[OptBootFileSize-13]
	_ = x[OptMeritDumpFile-14]
	_ = x[OptDomainName-15]
	_ = x[OptSwapServer-16]
	_ = x[OptRootPath-17]
	_ = x[OptExtensionFile-18]
	_ = x[OptIPLayerForwarding-19]
	_ = x[OptSrcrouteenabler-20]
	_ = x[OptPolicyFilter-21]
	_ = x[OptMaximumDGReassemblySize-22]
	_ = x[OptDefaultIPTTL-23]
	_ = x[OptPathMTUAgingTimeout-24]
	_ = x[OptMTUPlateau-25]
	_ = x[OptInterfaceMTUSize-26]
	_ = x[OptAllSubnetsAreLocal-27]
	_ = x[OptBroadcastAddress-28]
	_ = x[OptPerformMaskDiscovery-29]
	_ = x[OptProvideMasktoOthers-30]
	_ = x[OptPerformRouterDiscovery-31]
	_ = x[OptRouterSolicitationAddress-32]
	_ = x[OptStaticRoutingTable-33]
	_ = x[OptTrailerEncapsulation-34]
	_ = x[OptARPCacheTimeout-35]
	_ = x[OptEthernetEncapsulation-36]
	_ = x[OptDefaultTCPTimetoLive-37]
	_ = x[OptTCPKeepaliveInterval-38]
	_ = x[OptTCPKeepaliveGarbage-39]
	_ = x[OptNISDomainName-40]
	_ = x[OptNISServerAddresses-41]
	_ = x[OptNTPServersAddresses-42]
	_ = x[OptVendorSpecificInformation-43]
	_ = x[OptNetBIOSNameServer-44]
	_ = x[OptNetBIOSDatagramDistribution-45]
	_ = x[OptNetBIOSNodeType-46]
	_ = x[OptNetBIOSScope-47]
	_ = x[OptXWindowFontServer-48]
	_ = x[OptXWindowDisplayManager-49]
	_ = x[OptRequestedIPaddress-50]
	_ = x[OptIPAddressLeaseTime-51]
	_ = x[OptOptionOverload-52]
	_ = x[OptMessageType-53]
	_ = x[OptServerIdentification-54]
	_ = x[OptParameterRequestList-55]
	_ = x[OptMessage-56]
	_ = x[OptMaximumMessageSize-57]
	_ = x[OptRenewTimeValue-58]
	_ = x[OptRebindingTimeValue-59]
	_ = x[OptClientIdentifier-60]
	_ = x[OptClientIdentifier1-61]
}

const _OptNum_name = "WordAlignedSubnetMaskTimeOffsetRouterTimeServersNameServersDNSServersLogServersCookieServersLPRServersImpressServersRLPServersHostNameBootFileSizeMeritDumpFileDomainNameSwapServerRootPathExtensionFileIPLayerForwardingSrcrouteenablerPolicyFilterMaximumDGReassemblySizeDefaultIPTTLPathMTUAgingTimeoutMTUPlateauInterfaceMTUSizeAllSubnetsAreLocalBroadcastAddressPerformMaskDiscoveryProvideMasktoOthersPerformRouterDiscoveryRouterSolicitationAddressStaticRoutingTableTrailerEncapsulationARPCacheTimeoutEthernetEncapsulationDefaultTCPTimetoLiveTCPKeepaliveIntervalTCPKeepaliveGarbageNISDomainNameNISServerAddressesNTPServersAddressesVendorSpecificInformationNetBIOSNameServerNetBIOSDatagramDistributionNetBIOSNodeTypeNetBIOSScopeXWindowFontServerXWindowDisplayManagerRequestedIPaddressIPAddressLeaseTimeOptionOverloadMessageTypeServerIdentificationParameterRequestListMessageMaximumMessageSizeRenewTimeValueRebindingTimeValueClientIdentifierClientIdentifier1"

var _OptNum_index = [...]uint16{0, 11, 21, 31, 37, 48, 59, 69, 79, 92, 102, 116, 126, 134, 146, 159, 169, 179, 187, 200, 217, 232, 244, 267, 279, 298, 308, 324, 342, 358, 378, 397, 419, 444, 462, 482, 497, 518, 538, 558, 577, 590, 608, 627, 652, 669, 696, 711, 723, 740, 761, 779, 797, 811, 822, 842, 862, 869, 887, 901, 919, 935, 952}

func (i OptNum) String() string {
	if i >= OptNum(len(_OptNum_index)-1) {
		return "OptNum(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _OptNum_name[_OptNum_index[i]:_OptNum_index[i+1]]
}
