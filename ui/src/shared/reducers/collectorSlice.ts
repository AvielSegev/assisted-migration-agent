import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import type { CollectorStartRequest, Inventory } from '@generated/index';
import { CollectorStatusStatusEnum } from '@generated/index';
import { apiClient } from '@shared/api/client';

interface CollectorState {
  status: CollectorStatusStatusEnum;
  hasCredentials: boolean;
  error: string | null;
  inventory: Inventory | null;
  loading: boolean;
}

const initialState: CollectorState = {
  status: CollectorStatusStatusEnum.Ready,
  hasCredentials: false,
  error: null,
  inventory: null,
  loading: false,
};

export const fetchCollectorStatus = createAsyncThunk(
  'collector/fetchStatus',
  async () => {
    const response = await apiClient.getCollectorStatus();
    return response.data;
  }
);

export const startCollection = createAsyncThunk(
  'collector/start',
  async (credentials: CollectorStartRequest) => {
    const response = await apiClient.startCollector(credentials);
    return response.data;
  }
);

export const stopCollection = createAsyncThunk('collector/stop', async () => {
  await apiClient.stopCollector();
});

export const fetchInventory = createAsyncThunk(
  'collector/fetchInventory',
  async () => {
    const response = await apiClient.getInventory();
    return response.data;
  }
);

const collectorSlice = createSlice({
  name: 'collector',
  initialState,
  reducers: {
    setStatus: (state, action: PayloadAction<CollectorStatusStatusEnum>) => {
      state.status = action.payload;
    },
    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
    clearInventory: (state) => {
      state.inventory = null;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchCollectorStatus.pending, (state) => {
        state.loading = true;
      })
      .addCase(fetchCollectorStatus.fulfilled, (state, action) => {
        state.loading = false;
        if (action.payload) {
          state.status = action.payload.status;
          state.hasCredentials = action.payload.hasCredentials;
          state.error = action.payload.error ?? null;
        }
      })
      .addCase(fetchCollectorStatus.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message ?? 'Failed to fetch status';
      })
      .addCase(startCollection.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(startCollection.fulfilled, (state, action) => {
        state.loading = false;
        if (action.payload) {
          state.status = action.payload.status;
          state.hasCredentials = action.payload.hasCredentials;
          state.error = action.payload.error ?? null;
        }
      })
      .addCase(startCollection.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message ?? 'Failed to start collection';
      })
      .addCase(stopCollection.pending, (state) => {
        state.loading = true;
      })
      .addCase(stopCollection.fulfilled, (state) => {
        state.loading = false;
        state.status = CollectorStatusStatusEnum.Ready;
      })
      .addCase(stopCollection.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message ?? 'Failed to stop collection';
      })
      .addCase(fetchInventory.pending, (state) => {
        state.loading = true;
      })
      .addCase(fetchInventory.fulfilled, (state, action) => {
        state.loading = false;
        if (action.payload) {
          state.inventory = action.payload;
        }
      })
      .addCase(fetchInventory.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message ?? 'Failed to fetch inventory';
      });
  },
});

export const { setStatus, setError, clearInventory } = collectorSlice.actions;
export default collectorSlice.reducer;
