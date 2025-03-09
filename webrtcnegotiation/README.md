# WebRTC Negotiation

Handles the WebRTC negotiation for peer to peer/server (1:1) connections.

Follows perfect negotiation closely, however Pion does not support the perfect negotiation protocol at this time due to the lack of description rollback.

Instead if a offer collision occurs we drop both offers and renegotiate.
