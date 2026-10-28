import React, { useState, useEffect, useCallback, useRef } from "react";
import {
  View,
  Text,
  StyleSheet,
  ActivityIndicator,
  Pressable,
  Linking,
  Platform,
  Modal,
} from "react-native";
import { Map, Camera, Marker } from "@maplibre/maplibre-react-native";
import * as Location from "expo-location";

import {
  fetchNearbyTerminals,
  fetchRouteData,
  Terminal,
} from "./services/terminalService";
import { Colors, Spacing, Radius, Typography } from "./theme";

const MAP_STYLE = "https://tiles.openfreemap.org/styles/liberty";

const DEFAULT_LOCATION = {
  latitude: 9.03,
  longitude: 38.74,
};

interface RouteInfo {
  averageWaitMinutes: number;
  averageQueueLength: number;
  totalReports: number;
  distanceMeters: number;
  fromTerminalName: string;
  toTerminalName: string;
}

const RADII = [2000, 5000, 10000];

export default function App() {
  const [location, setLocation] = useState(DEFAULT_LOCATION);
  const [terminals, setTerminals] = useState<Terminal[]>([]);
  const [selectedTerminal, setSelectedTerminal] = useState<Terminal | null>(
    null,
  );

  const [loading, setLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [deniedPermission, setDeniedPermission] = useState(false);
  const [servicesDisabled, setServicesDisabled] = useState(false);
  const [searchRadius, setSearchRadius] = useState(2000);

  const cameraRef = useRef<any>(null);

  const loadMapData = useCallback(async (radiusToSearch = 2000) => {
    setLoading(true);
    setErrorMsg(null);
    setDeniedPermission(false);
    setServicesDisabled(false);

    try {
      let activeCoords = DEFAULT_LOCATION;

      const permission = await Location.requestForegroundPermissionsAsync();

      if (permission.status !== "granted") {
        setDeniedPermission(true);
        setErrorMsg(
          "Location permission is required to find nearby taxi terminals.",
        );
        setLoading(false);
        return;
      }

      const enabled = await Location.hasServicesEnabledAsync();

      if (!enabled) {
        setServicesDisabled(true);
        setErrorMsg(
          "Your device location services are disabled. Enable them to use your current location.",
        );
        setLoading(false);
        return;
      }

      try {
        if (Platform.OS === "android") {
          await Location.enableNetworkProviderAsync().catch(() => {});
        }

        const lastLocation = await Location.getLastKnownPositionAsync();

        if (lastLocation) {
          activeCoords = {
            latitude: lastLocation.coords.latitude,
            longitude: lastLocation.coords.longitude,
          };
        }

        try {
          const currentLocation = await Location.getCurrentPositionAsync({
            accuracy: Location.Accuracy.Balanced,
          });

          activeCoords = {
            latitude: currentLocation.coords.latitude,
            longitude: currentLocation.coords.longitude,
          };
        } catch (locationError) {
          // Use the last known location if a fresh GPS fix is unavailable.
          console.warn(
            "Could not obtain a fresh location; using the available fallback.",
            locationError,
          );
        }
      } catch (locationError) {
        console.warn("Location lookup failed:", locationError);
      }

      setLocation(activeCoords);
      setSearchRadius(radiusToSearch);

      cameraRef.current?.setStop?.({
        center: [activeCoords.longitude, activeCoords.latitude],
        zoom: 12,
        animationDuration: 800,
      });

      const nearbyTerminals = await fetchNearbyTerminals(
        activeCoords.latitude,
        activeCoords.longitude,
        radiusToSearch,
      );

      setTerminals(nearbyTerminals ?? []);
    } catch (error) {
      console.error("Failed to load map data:", error);
      setErrorMsg(
        "Unable to load nearby taxi terminals. Check your connection and try again.",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  const [selectedOrigin, setSelectedOrigin] = useState<string | null>(null);
  const [selectedDestination, setSelectedDestination] = useState<string | null>(
    null,
  );
  const [routeInfo, setRouteInfo] = useState<RouteInfo | null>(null);

  {/* Select an origin terminal from the nearby list */}
  <View style={styles.terminalSelector}>
    {terminals.map((terminal) => (
      <Pressable
        key={terminal.id}
        style={styles.selectorItem}
        onPress={() => setSelectedOrigin(terminal.id)}
      >
        <Text>{terminal.name}</Text>
      </Pressable>
    ))}
  </View>

  useEffect(() => {
    void loadMapData(2000);
  }, [loadMapData]);

  const originName = terminals.find((t) => t.id === selectedOrigin)?.name ?? "";
  const destinationName =
    terminals.find((t) => t.id === selectedDestination)?.name ?? "";

  // Fetch route info when both origin and destination are selected
  useEffect(() => {
    if (!selectedOrigin || !selectedDestination) {
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        const data = await fetchRouteData(selectedOrigin, selectedDestination);
        if (cancelled) return;

        const origin = terminals.find((t) => t.id === selectedOrigin);
        const destination = terminals.find((t) => t.id === selectedDestination);

        setRouteInfo({
          averageWaitMinutes: data.avarageWaitMinutes,
          averageQueueLength: data.avarageQueueLenght,
          totalReports: data.totoalReports,
          distanceMeters: 0,
          fromTerminalName: origin?.name ?? "Unknown",
          toTerminalName: destination?.name ?? "Unknown",
        });
      } catch (error) {
        if (cancelled) return;
        console.error("Failed to fetch route data:", error);
        setRouteInfo(null);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [selectedOrigin, selectedDestination, terminals]);

  const handleRadiusChange = (radius: number) => {
    void loadMapData(radius);
  };

  const openSettings = () => {
    void Linking.openSettings();
  };

  const formatRadius = (radius: number) => {
    return `${radius / 1000} km`;
  };

  const renderActionButton = (label: string, onPress: () => void) => (
    <Pressable style={styles.actionButton} onPress={onPress}>
      <Text style={styles.actionButtonText}>{label}</Text>
    </Pressable>
  );

  return (
    <View style={styles.container}>
      {/* Map */}
      <Map style={styles.map} mapStyle={MAP_STYLE} attribution logo>
        <Camera
          ref={cameraRef}
          initialViewState={{
            center: [location.longitude, location.latitude],
            zoom: 12,
          }}
        />

        {/* User's current location: blue dot */}
        <Marker
          id="user-location"
          lngLat={[location.longitude, location.latitude]}
        >
          <View style={styles.userLocationOuter}>
            <View style={styles.userLocationInner} />
          </View>
        </Marker>

        {/* Nearby taxi terminals */}
        {terminals.map((terminal) => (
          <Marker
            key={terminal.id}
            id={terminal.id}
            lngLat={[terminal.longitude, terminal.latitude]}
            onPress={() => setSelectedTerminal(terminal)}
          >
            <Pressable
              onPress={() => setSelectedTerminal(terminal)}
              style={styles.markerContainer}
            >
              <View style={styles.marker}>
                <Text style={styles.markerText}>🚕</Text>
              </View>

              <View style={styles.markerLabel}>
                <Text style={styles.markerLabelText} numberOfLines={1}>
                  {terminal.name}
                </Text>
              </View>
            </Pressable>
          </Marker>
        ))}
      </Map>

      {/* Top information panel */}
      <View style={styles.topPanel}>
        <View style={styles.headerRow}>
          <View style={styles.headerTextContainer}>
            <Text style={styles.appTitle}>Taxi Connect</Text>
            <Text style={styles.subtitle}>Find nearby taxi terminals</Text>
          </View>

          <Pressable
            style={styles.refreshButton}
            onPress={() => void loadMapData(searchRadius)}
            disabled={loading}
          >
            {loading ? (
              <ActivityIndicator size="small" color={Colors.primary} />
            ) : (
              <Text style={styles.refreshText}>↻ Refresh</Text>
            )}
          </Pressable>
        </View>

        <View style={styles.terminalCountContainer}>
          <Text style={styles.terminalCount}>{terminals.length}</Text>
          <Text style={styles.terminalCountLabel}>
            {terminals.length === 1 ? "terminal found" : "terminals found"}
          </Text>
        </View>

        <Text style={styles.radiusTitle}>Search radius</Text>

        <View style={styles.radiusRow}>
          {RADII.map((radius) => {
            const isSelected = searchRadius === radius;

            return (
              <Pressable
                key={radius}
                style={[
                  styles.radiusButton,
                  isSelected && styles.radiusButtonSelected,
                ]}
                onPress={() => handleRadiusChange(radius)}
                disabled={loading}
              >
                <Text
                  style={[
                    styles.radiusButtonText,
                    isSelected && styles.radiusButtonTextSelected,
                  ]}
                >
                  {formatRadius(radius)}
                </Text>
              </Pressable>
            );
          })}
        </View>
      </View>

      {/* Route selection: chosen origin and destination */}
      {(selectedOrigin || selectedDestination) && (
        <View style={styles.routePanel}>
          <Text style={styles.routeLabel}>Route</Text>
          <View style={styles.routeRow}>
            <Text style={styles.routeStop}>{originName}</Text>
            <Text style={styles.routeArrow}>&#8594;</Text>
            <Text style={styles.routeStop}>{destinationName}</Text>
          </View>
        </View>
      )}

      {/* Route info: estimated wait and queue stats for the chosen route */}
      {routeInfo && (
        <View style={styles.routeInfoCard}>
          <Text style={styles.routeTitle}>
            {routeInfo.fromTerminalName} &rarr; {routeInfo.toTerminalName}
          </Text>
          <View style={styles.routeStats}>
            <View style={styles.statItem}>
              <Text style={styles.statLabel}>Estimated wait</Text>
              <Text style={styles.statValue}>
                {routeInfo.averageWaitMinutes} min
              </Text>
            </View>
            <View style={styles.statItem}>
              <Text style={styles.statLabel}>Queue length</Text>
              <Text style={styles.statValue}>
                {routeInfo.averageQueueLength}
              </Text>
            </View>
            <View style={styles.statItem}>
              <Text style={styles.statLabel}>Reports</Text>
              <Text style={styles.statValue}>{routeInfo.totalReports}</Text>
            </View>
          </View>
        </View>
      )}

      {/* Loading indicator */}
      {loading && (
        <View style={styles.loadingPanel}>
          <ActivityIndicator size="small" color={Colors.primary} />
          <Text style={styles.loadingText}>Finding nearby terminals...</Text>
        </View>
      )}


      {/* Error and permission messages */}
      {!loading && errorMsg && (
        <View style={styles.errorPanel}>
          <Text style={styles.errorTitle}>
            {deniedPermission
              ? "Location permission needed"
              : servicesDisabled
                ? "Location is turned off"
                : "Something went wrong"}
          </Text>

          <Text style={styles.errorText}>{errorMsg}</Text>

          {deniedPermission || servicesDisabled
            ? renderActionButton("Open Settings", openSettings)
            : renderActionButton(
                "Try Again",
                () => void loadMapData(searchRadius),
              )}
        </View>
      )}

      {/* Empty results */}
      {!loading && !errorMsg && terminals.length === 0 && (
        <View style={styles.emptyPanel}>
          <Text style={styles.emptyTitle}>No terminals found</Text>
          <Text style={styles.emptyText}>
            Try increasing the search radius to find terminals farther away.
          </Text>
        </View>
      )}

      {/* Current location button */}
      <Pressable
        style={styles.myLocationButton}
        onPress={() => {
          cameraRef.current?.setStop?.({
            center: [location.longitude, location.latitude],
            zoom: 14,
            animationDuration: 800,
          });
        }}
      >
        <Text style={styles.myLocationIcon}>◎</Text>
        <Text style={styles.myLocationText}>My location</Text>
      </Pressable>

      {/* Selected terminal details */}
      <Modal
        visible={selectedTerminal !== null}
        transparent
        animationType="slide"
        onRequestClose={() => setSelectedTerminal(null)}
      >
        <Pressable
          style={styles.modalOverlay}
          onPress={() => setSelectedTerminal(null)}
        >
          <Pressable style={styles.modalContent} onPress={() => {}}>
            <View style={styles.modalHandle} />

            <View style={styles.modalHeader}>
              <View style={styles.modalIcon}>
                <Text style={styles.modalTaxiIcon}>🚕</Text>
              </View>

              <View style={styles.modalTitleContainer}>
                <Text style={styles.modalTitle}>
                  {selectedTerminal?.name ?? "Taxi Terminal"}
                </Text>
                <Text style={styles.modalSubtitle}>Taxi terminal details</Text>
              </View>

              <Pressable
                onPress={() => setSelectedTerminal(null)}
                style={styles.closeButton}
              >
                <Text style={styles.closeButtonText}>✕</Text>
              </Pressable>
            </View>

            <View style={styles.divider} />

            <Text style={styles.detailLabel}>Latitude</Text>
            <Text style={styles.detailValue}>
              {selectedTerminal?.latitude.toFixed(6)}
            </Text>

            <Text style={styles.detailLabel}>Longitude</Text>
            <Text style={styles.detailValue}>
              {selectedTerminal?.longitude.toFixed(6)}
            </Text>

            <Pressable
              style={styles.modalButton}
              onPress={() => {
                if (!selectedTerminal) return;

                cameraRef.current?.setStop?.({
                  center: [
                    selectedTerminal.longitude,
                    selectedTerminal.latitude,
                  ],
                  zoom: 15,
                  animationDuration: 800,
                });

                setSelectedTerminal(null);
              }}
            >
              <Text style={styles.modalButtonText}>Show on map</Text>
            </Pressable>
          </Pressable>
        </Pressable>
      </Modal>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.background,
  },

  map: {
    flex: 1,
  },

  topPanel: {
    position: "absolute",
    top: 48,
    left: Spacing.md,
    right: Spacing.md,
    backgroundColor: Colors.surface,
    borderRadius: Radius.lg,
    padding: Spacing.md,
    elevation: 5,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.15,
    shadowRadius: 5,
  },

  headerRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
  },

  headerTextContainer: {
    flex: 1,
  },

  appTitle: {
    ...Typography.h1,
    color: Colors.primaryText,
    fontWeight: "700",
    fontSize: 22,
  },

  subtitle: {
    ...Typography.body,
    color: Colors.secondaryText,
    marginTop: 3,
  },

  refreshButton: {
    paddingHorizontal: Spacing.sm,
    paddingVertical: Spacing.sm,
    borderRadius: Radius.md,
    backgroundColor: Colors.background,
  },

  refreshText: {
    color: Colors.primary,
    fontWeight: "600",
  },

  terminalCountContainer: {
    flexDirection: "row",
    alignItems: "baseline",
    marginTop: Spacing.md,
    gap: 6,
  },

  terminalCount: {
    fontSize: 24,
    fontWeight: "700",
    color: Colors.primary,
  },

  terminalCountLabel: {
    ...Typography.body,
    color: Colors.secondaryText,
  },

  radiusTitle: {
    ...Typography.body,
    color: Colors.primaryText,
    fontWeight: "600",
    marginTop: Spacing.md,
    marginBottom: Spacing.sm,
  },

  radiusRow: {
    flexDirection: "row",
    gap: Spacing.sm,
  },

  radiusButton: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    paddingVertical: 10,
    borderRadius: Radius.md,
    borderWidth: 1,
    borderColor: Colors.border,
    backgroundColor: Colors.background,
  },

  radiusButtonSelected: {
    backgroundColor: Colors.primary,
    borderColor: Colors.primary,
  },

  radiusButtonText: {
    fontSize: 13,
    color: Colors.primaryText,
    fontWeight: "600",
  },

  radiusButtonTextSelected: {
    color: "#FFFFFF",
  },

  // Blue dot representing the user's current location
  userLocationOuter: {
    width: 28,
    height: 28,
    borderRadius: 14,
    backgroundColor: "rgba(66, 133, 244, 0.20)",
    alignItems: "center",
    justifyContent: "center",
  },

  userLocationInner: {
    width: 16,
    height: 16,
    borderRadius: 8,
    backgroundColor: "#4285F4",
    borderWidth: 3,
    borderColor: "#FFFFFF",
    elevation: 5,
    shadowColor: "#000000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.3,
    shadowRadius: 3,
  },

  // Taxi terminal markers
  markerContainer: {
    alignItems: "center",
  },

  marker: {
    width: 38,
    height: 38,
    borderRadius: 19,
    backgroundColor: Colors.surface,
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 2,
    borderColor: Colors.primary,
    elevation: 4,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.2,
    shadowRadius: 3,
  },

  markerText: {
    fontSize: 20,
  },

  markerLabel: {
    maxWidth: 130,
    marginTop: 3,
    paddingHorizontal: 7,
    paddingVertical: 3,
    backgroundColor: Colors.surface,
    borderRadius: Radius.sm,
    elevation: 2,
  },

  markerLabelText: {
    fontSize: 11,
    fontWeight: "600",
    color: Colors.primaryText,
  },

  loadingPanel: {
    position: "absolute",
    bottom: 100,
    alignSelf: "center",
    flexDirection: "row",
    alignItems: "center",
    gap: Spacing.sm,
    backgroundColor: Colors.surface,
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.sm,
    borderRadius: Radius.lg,
    elevation: 4,
  },

  loadingText: {
    ...Typography.body,
    color: Colors.primaryText,
  },

  errorPanel: {
    position: "absolute",
    bottom: 100,
    left: Spacing.md,
    right: Spacing.md,
    padding: Spacing.md,
    backgroundColor: Colors.surface,
    borderRadius: Radius.lg,
    elevation: 4,
  },

  errorTitle: {
    fontSize: 16,
    fontWeight: "700",
    color: Colors.primaryText,
    marginBottom: Spacing.xs,
  },

  errorText: {
    ...Typography.body,
    color: Colors.secondaryText,
    marginBottom: Spacing.md,
  },

  actionButton: {
    alignSelf: "flex-start",
    backgroundColor: Colors.primary,
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.sm,
    borderRadius: Radius.md,
  },

  actionButtonText: {
    color: "#FFFFFF",
    fontWeight: "600",
  },

  emptyPanel: {
    position: "absolute",
    bottom: 100,
    left: Spacing.md,
    right: Spacing.md,
    padding: Spacing.md,
    backgroundColor: Colors.surface,
    borderRadius: Radius.lg,
    elevation: 4,
  },

  emptyTitle: {
    fontSize: 16,
    fontWeight: "700",
    color: Colors.primaryText,
    marginBottom: Spacing.xs,
  },

  emptyText: {
    ...Typography.body,
    color: Colors.secondaryText,
  },

  myLocationButton: {
    position: "absolute",
    right: Spacing.md,
    bottom: 32,
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    paddingHorizontal: Spacing.md,
    paddingVertical: Spacing.sm,
    borderRadius: Radius.lg,
    backgroundColor: Colors.surface,
    elevation: 5,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.15,
    shadowRadius: 4,
  },

  myLocationIcon: {
    fontSize: 24,
    color: "#4285F4",
    fontWeight: "700",
  },

  myLocationText: {
    color: Colors.primaryText,
    fontWeight: "600",
  },

  modalOverlay: {
    flex: 1,
    justifyContent: "flex-end",
    backgroundColor: "rgba(0, 0, 0, 0.4)",
  },

  modalContent: {
    backgroundColor: Colors.surface,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    padding: Spacing.lg,
    paddingBottom: 36,
  },

  modalHandle: {
    alignSelf: "center",
    width: 40,
    height: 5,
    borderRadius: 3,
    backgroundColor: Colors.border,
    marginBottom: Spacing.lg,
  },

  modalHeader: {
    flexDirection: "row",
    alignItems: "center",
  },

  modalIcon: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: Colors.background,
    alignItems: "center",
    justifyContent: "center",
    marginRight: Spacing.md,
  },

  modalTaxiIcon: {
    fontSize: 24,
  },

  modalTitleContainer: {
    flex: 1,
  },

  modalTitle: {
    fontSize: 18,
    fontWeight: "700",
    color: Colors.primaryText,
  },

  modalSubtitle: {
    ...Typography.body,
    color: Colors.secondaryText,
    marginTop: 3,
  },

  closeButton: {
    padding: Spacing.sm,
  },

  closeButtonText: {
    fontSize: 18,
    color: Colors.secondaryText,
  },

  divider: {
    height: 1,
    backgroundColor: Colors.border,
    marginVertical: Spacing.lg,
  },

  detailLabel: {
    ...Typography.body,
    color: Colors.secondaryText,
    marginBottom: 4,
  },

  detailValue: {
    fontSize: 15,
    fontWeight: "600",
    color: Colors.primaryText,
    marginBottom: Spacing.md,
  },

  modalButton: {
    alignItems: "center",
    paddingVertical: Spacing.md,
    borderRadius: Radius.md,
    backgroundColor: Colors.primary,
    marginTop: Spacing.sm,
  },

  modalButtonText: {
    color: "#FFFFFF",
    fontSize: 15,
    fontWeight: "700",
  },

  selectorItem: {
    padding: 12,
    borderRadius: 8,
    margin: 4,
    backgroundColor: Colors.surface,
    elevation: 2,
  },

  terminalSelector: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: Spacing.sm,
    paddingHorizontal: Spacing.md,
    marginVertical: Spacing.sm,
  },

  routeDetails: {
    padding: Spacing.md,
    backgroundColor: Colors.surface,
    borderRadius: Radius.md,
    marginVertical: Spacing.md,
  },

  routeHeader: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    paddingBottom: Spacing.xs,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderColor: Colors.border,
  },

  routeTitle: {
    ...Typography.h2,
    color: Colors.primaryText,
  },

  closeBtn: {
    padding: Spacing.xs,
    color: Colors.error,
  },

  routeStats: {
    flexDirection: "row",
    justifyContent: "space-around",
    marginVertical: Spacing.md,
  },

  statItem: {
    alignItems: "center",
    padding: 8,
    borderRadius: 8,
    backgroundColor: Colors.background,
    margin: 4,
  },

  statLabel: {
    ...Typography.caption,
    color: Colors.secondaryText,
    marginBottom: 4,
  },

  statValue: {
    ...Typography.body,
    color: Colors.primaryText,
  },
  routeInfo: {
    padding: Spacing.md,
    backgroundColor: Colors.surface,
    borderRadius: Radius.md,
    marginVertical: Spacing.md,
  },

  alternativesSection: {
    padding: Spacing.md,
    backgroundColor: Colors.surface,
    borderRadius: Radius.md,
    marginVertical: Spacing.md,
  },

  routePanel: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    paddingHorizontal: Spacing.md,
    marginTop: Spacing.sm,
    backgroundColor: Colors.surface,
    borderRadius: Radius.md,
    alignSelf: "center",
    width: "85%",
  },

  routeLabel: {
    ...Typography.caption,
    color: Colors.secondaryText,
  },

  routeRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    flex: 1,
    flexWrap: "wrap",
  },

  routeStop: {
    ...Typography.body,
    color: Colors.primaryText,
    fontWeight: "600",
  },

  routeArrow: {
    color: Colors.primary,
    fontWeight: "700",
  },

  routeInfoCard: {
    padding: Spacing.md,
    backgroundColor: Colors.surface,
    borderRadius: Radius.md,
    marginHorizontal: Spacing.md,
    marginVertical: Spacing.sm,
    alignSelf: "center",
    width: "85%",
    elevation: 2,
  },

  alternativeItem: {
    padding: 12,
    borderRadius: 8,
    backgroundColor: Colors.background,
    margin: 4,
    elevation: 2,
  },
});

