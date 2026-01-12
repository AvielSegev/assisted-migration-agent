import { combineReducers } from '@reduxjs/toolkit';
import collectorReducer from './collectorSlice';
import agentReducer from './agentSlice';
import vmReducer from './vmSlice';

const rootReducer = combineReducers({
  collector: collectorReducer,
  agent: agentReducer,
  vm: vmReducer,
});

export type RootState = ReturnType<typeof rootReducer>;
export default rootReducer;

export * from './collectorSlice';
export * from './agentSlice';
export * from './vmSlice';
