import { combineReducers } from '@reduxjs/toolkit';
import collectorReducer from './collectorSlice';
import agentReducer from './agentSlice';

const rootReducer = combineReducers({
  collector: collectorReducer,
  agent: agentReducer,
});

export type RootState = ReturnType<typeof rootReducer>;
export default rootReducer;

export * from './collectorSlice';
export * from './agentSlice';
