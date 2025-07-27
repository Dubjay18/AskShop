package contracts

// USER EVENTS (user.event.*)
const (
	UserEventRegistered = "user.event.registered"
	UserEventUpdated    = "user.event.updated"
)

// PRODUCT EVENTS (product.event.*)
const (
	ProductEventCreated = "product.event.created"
	ProductEventUpdated = "product.event.updated"
	ProductEventDeleted = "product.event.deleted"
)

// CART COMMANDS (cart.cmd.*)
const (
	CartCmdAddItem    = "cart.cmd.add_item"
	CartCmdRemoveItem = "cart.cmd.remove_item"
	CartCmdClear      = "cart.cmd.clear"
)

// ORDER EVENTS (order.event.*)
const (
	OrderEventPlaced     = "order.event.placed"
	OrderEventConfirmed  = "order.event.confirmed"
	OrderEventCancelled  = "order.event.cancelled"
	OrderEventDispatched = "order.event.dispatched"
)

// AI COMMANDS (ai.cmd.*)
const (
	AICmdExplainProduct = "ai.cmd.explain_product"
)

// NOTIFICATION EVENTS (notify.event.*)
const (
	NotifyEventOrderConfirmed = "notify.event.order_confirmed"
	NotifyEventOrderCancelled = "notify.event.order_cancelled"
)

// NOTIFICATION COMMANDS (notify.cmd.*)
const (
	NotifyCmdSendEmail = "notify.cmd.send_email"
	NotifyCmdSendSMS   = "notify.cmd.send_sms"
)
