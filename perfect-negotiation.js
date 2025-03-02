/**
 * @param {Object} options
 * @param {WebSocket} options.signalServer Websocket signalling server
 * @param {boolean} options.isPolite One peer should be polite and the other impolite
 * @param {boolean} options.peerConnection If true, the peer is a viewer and will not send any offers
 * @param {RTCOfferOptions | undefined} options.offerOptions Offer options
 * @returns {RTCPeerConnection}
 */
const perfectNegotiation = async ({
  signalServer,
  isPolite = true,
  peerConnection = new RTCPeerConnection(),
  offerOptions,
} = {}) => {
  let makingOffer = false;

  const sendOffer = async () => {
    if (signalServer.readyState !== WebSocket.OPEN) {
      console.log("Signal server not open");
      return;
    }
    try {
      makingOffer = true;
      await peerConnection.setLocalDescription(offerOptions);
      signalServer.send(
        JSON.stringify({ description: peerConnection.localDescription }),
      );
    } catch (err) {
      console.error(err);
    } finally {
      makingOffer = false;
    }
  };

  // Handle negotiation
  peerConnection.onnegotiationneeded = async () => {
    console.log("Negotiation needed");
    await sendOffer();
  };

  // Handle incoming messages from signaling server
  signalServer.onmessage = async ({ data }) => {
    const { description, candidate, retry } = JSON.parse(data);
    try {
      if (description) {
        const offerCollision =
          description.type === "offer" &&
          (makingOffer || peerConnection.signalingState !== "stable");

        if (offerCollision) {
          if (isPolite) {
            console.log(
              "Offer collision detected: Polite peer rolls back and accepts remote offer.",
            );
            // Rollback any pending local description
            await Promise.all([
              peerConnection.setLocalDescription({ type: "rollback" }),
              peerConnection.setRemoteDescription(description)
            ]);
            await peerConnection.setLocalDescription();
            signalServer.send(
              JSON.stringify({ description: peerConnection.localDescription }),
            );
          } else {
            console.log(
              "Offer collision detected: Impolite peer ignores and proceeds.",
            );
            return;
          }
        } else {
          await peerConnection.setRemoteDescription(description);
          if (description.type === "offer") {
            await peerConnection.setLocalDescription();
            signalServer.send(
              JSON.stringify({ description: peerConnection.localDescription }),
            );
          }
        }
      } else if (candidate) {
        console.log("Adding ice candidate", candidate);
        await peerConnection.addIceCandidate(candidate);
      }
    } catch (err) {
      console.error(err);
    }
  };

  return {
    sendOffer,
    peerConnection,
  };
};

export default perfectNegotiation;
