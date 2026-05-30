package render

const (
	diskAwaitHistoryCapMS   = 50
	diskQueueHistoryCap     = 8
	diskHistoryCritical     = 100
	diskHistoryWarn         = 30
	diskHistoryAcceptable   = 10
	diskAwaitCriticalMS     = 50
	diskAwaitWarnMS         = 15
	diskAwaitAcceptableMS   = 5
	diskQueueCriticalDepth  = 8
	diskQueueWarnDepth      = 4
	diskUtilCriticalPercent = 90
	diskUtilWarnPercent     = 60
	diskUtilOKPercent       = 40

	netIssueDropWeight    = 10
	netIssueErrorWeight   = 20
	netIssueCritical      = 50
	netIssueWarn          = 20
	linkBytesPerMegabit   = 125000
	tcpRetransCriticalPPS = 100
	tcpRetransWarnPPS     = 10
	tcpResetCriticalPPS   = 10

	utilCriticalPercent = 95
	utilWarnPercent     = 80
	utilOKPercent       = 40

	memoryCriticalPercent = 90
	memoryWarnPercent     = 85
	memoryOKPercent       = 60

	availabilityCriticalPercent = 5
	availabilityWarnPercent     = 15
	availabilityInfoPercent     = 35

	psiCriticalPercent = 20
	psiWarnPercent     = 5

	temperatureCriticalPercent = 80
	temperatureWarnPercent     = 70
	temperatureOKPercent       = 60

	powerCriticalPercent = 98
	powerWarnPercent     = 90
	powerOKPercent       = 65

	severityRankCritical = 3
	severityRankWarn     = 2

	sparklineRuneByteBudget = 3
)
