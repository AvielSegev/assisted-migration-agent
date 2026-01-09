import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import {
  AgentModeRequestModeEnum,
  AgentStatusModeEnum,
  AgentStatusConsoleConnectionEnum,
} from '@generated/index';
import { apiClient } from '@shared/api/client';

interface AgentState {
  mode: AgentStatusModeEnum;
  consoleConnection: AgentStatusConsoleConnectionEnum;
  loading: boolean;
  error: string | null;
}

const initialState: AgentState = {
  mode: AgentStatusModeEnum.Disconnected,
  consoleConnection: AgentStatusConsoleConnectionEnum.Disconnected,
  loading: false,
  error: null,
};

export const fetchAgentStatus = createAsyncThunk(
  'agent/fetchStatus',
  async () => {
    const response = await apiClient.getAgentStatus();
    return response.data;
  }
);

export const changeAgentMode = createAsyncThunk(
  'agent/changeMode',
  async (mode: AgentModeRequestModeEnum) => {
    const response = await apiClient.setAgentMode({ mode });
    return response.data;
  }
);

const agentSlice = createSlice({
  name: 'agent',
  initialState,
  reducers: {
    setMode: (state, action: PayloadAction<AgentStatusModeEnum>) => {
      state.mode = action.payload;
    },
    setConsoleConnection: (
      state,
      action: PayloadAction<AgentStatusConsoleConnectionEnum>
    ) => {
      state.consoleConnection = action.payload;
    },
    setAgentError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchAgentStatus.pending, (state) => {
        state.loading = true;
      })
      .addCase(fetchAgentStatus.fulfilled, (state, action) => {
        state.loading = false;
        if (action.payload) {
          state.mode = action.payload.mode;
          state.consoleConnection = action.payload.console_connection;
          state.error = null;
        }
      })
      .addCase(fetchAgentStatus.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message ?? 'Failed to fetch agent status';
      })
      .addCase(changeAgentMode.pending, (state) => {
        state.loading = true;
      })
      .addCase(changeAgentMode.fulfilled, (state, action) => {
        state.loading = false;
        if (action.payload) {
          state.mode = action.payload.mode;
          state.consoleConnection = action.payload.console_connection;
          state.error = null;
        }
      })
      .addCase(changeAgentMode.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message ?? 'Failed to change agent mode';
      });
  },
});

export const { setMode, setConsoleConnection, setAgentError } =
  agentSlice.actions;
export default agentSlice.reducer;
