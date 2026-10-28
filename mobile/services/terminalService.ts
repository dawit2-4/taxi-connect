export interface Terminal {
  id: string;
  name: string;
  latitude: number;
  longitude: number;
}

export async function fetchNearbyTerminals(
  latitude: number,
  longitude: number,
  radius: number = 2000,
): Promise<Terminal[]> {
  const lat = Number(latitude);
  const lon = Number(longitude);
  const rad = Number(radius);

  if (!Number.isFinite(lat) || lat < -90 || lat > 90) {
    throw new Error(`Invalid latitude: "${latitude}".`);
  }

  if (!Number.isFinite(lon) || lon < -180 || lon > 180) {
    throw new Error(`Invalid longitude: "${longitude}".`);
  }

  if (!Number.isFinite(rad) || rad <= 0) {
    throw new Error(`Invalid radius: "${radius}".`);
  }

  const params = new URLSearchParams({
    lat: lat.toFixed(6),
    lon: lon.toFixed(6),
    radius: Math.round(rad).toString(),
  });

  // Use your computer's LAN IP when testing on your physical phone.
  const url = `http://192.168.100.31:8080/terminals/near?${params.toString()}`;

  console.log("Fetching terminals:", url);

  try {
    const response = await fetch(url, {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      const errorBody = await response.text();
      throw new Error(`HTTP ${response.status}: ${errorBody}`);
    }

    const data: unknown = await response.json();

    if (!Array.isArray(data)) {
      console.warn("Unexpected terminal response:", data);
      return [];
    }

    const terminals: Terminal[] = data.map((terminal: any) => {
      let latitude = Number(terminal.latitude ?? terminal.lat ?? 0);
      let longitude = Number(terminal.longitude ?? terminal.lon ?? 0);

      // Supports PostGIS WKT: POINT(longitude latitude)
      if (typeof terminal.location === "string") {
        const match = terminal.location.match(
          /POINT\s*\(\s*(-?\d+(?:\.\d+)?)\s+(-?\d+(?:\.\d+)?)\s*\)/i,
        );

        if (match) {
          longitude = Number(match[1]);
          latitude = Number(match[2]);
        }
      }

      return {
        id: String(terminal.id),
        name: terminal.name ?? "Unknown Terminal",
        latitude,
        longitude,
      };
    });

    // Ignore invalid coordinates before rendering map markers.
    const validTerminals = terminals.filter(
      (terminal) =>
        Number.isFinite(terminal.latitude) &&
        terminal.latitude >= -90 &&
        terminal.latitude <= 90 &&
        Number.isFinite(terminal.longitude) &&
        terminal.longitude >= -180 &&
        terminal.longitude <= 180,
    );

    console.log(`Fetched ${validTerminals.length} valid terminal(s).`);

    return validTerminals;
  } catch (error) {
    console.error("Error fetching nearby terminals:", error);
    throw error;
  }
}

export async function fetchRouteData(
  fromTerminalId: string,
  toTerminalId: string,
): Promise<{
  avarageWaitMinutes: number;
  avarageQueueLenght: number;
  totoalReports: number;
}> {
  const url = `http://192.168.100.31:8080/queues/route?from=${fromTerminalId}&to=${toTerminalId}`;

  const response = await fetch(url, {
    headers: { Accept: "application/json" },
  });

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  const data = await response.json();

  return {
    avarageWaitMinutes: data.avarageWaitMinutes || 0,
    avarageQueueLenght: data.avarageQueueLenght || 0,
    totoalReports: data.totoalReports || 0,
  };
}
