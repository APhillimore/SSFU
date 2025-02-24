/**
 * @param {Object} options
 * @param {WebSocket} options.signalServer Websocket signalling server
 * @param {boolean} options.isPolite One peer should be polite and the other impolite
 * @param {boolean} options.peerConnection If true, the peer is a viewer and will not send any offers
 * @returns {RTCPeerConnection}
 */
const perfectNegotiation = async ({
  signalServer,
  isPolite = true,
  peerConnection = new RTCPeerConnection(),
} = {}) => {
  let makingOffer = false;
  // Handle negotiation
  peerConnection.onnegotiationneeded = async () => {
    try {
      makingOffer = true;
      await peerConnection.setLocalDescription();
      signalServer.send(
        JSON.stringify({ description: peerConnection.localDescription }),
      );
    } catch (err) {
      console.error(err);
    } finally {
      makingOffer = false;
    }
  };

  // Handle incoming messages from signaling server
  signalServer.onmessage = async ({ description, candidate }) => {
    try {
      if (description) {
        const offerCollision =
          description.type === "offer" &&
          (makingOffer || peerConnection.signalingState !== "stable");

        if (offerCollision) {
          if (isPolite) {
            console.log(
              "Offer collision detected: Polite peer defers to remote offer.",
            );
            makingOffer = false;
            await peerConnection.setRemoteDescription(description);
            await peerConnection.setLocalDescription();
            signalServer.send(
              JSON.stringify({ description: peerConnection.localDescription }),
            );
          } else {
            console.log(
              "Offer collision detected: Impolite peer ignores and proceeds.",
            );
            // The impolite peer just continues with its own offer.
          }
          return;
        }

        await peerConnection.setRemoteDescription(description);
        if (description.type === "offer") {
          await peerConnection.setLocalDescription();
          signalServer.send(
            JSON.stringify({ description: peerConnection.localDescription }),
          );
        }
      } else if (candidate) {
        await peerConnection.addIceCandidate(candidate);
      }
    } catch (err) {
      console.error(err);
    }
  };

  return peerConnection;
};

export default perfectNegotiation;
