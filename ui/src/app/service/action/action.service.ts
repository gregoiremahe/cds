import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { Action, PipelineUsingAction } from '../../model/action.model';

/**
 * Service to access Public Action
 */
@Injectable()
export class ActionService {

    constructor(private _http: HttpClient) { }

    getActions(): Observable<Action[]> {
        return this._http.get<Action[]>('/action');
    }

    get(groupName: string, name: string): Observable<Action> {
        return this._http.get<Action>(`/action/${groupName}/${name}`);
    }

    /**
     * Get action usage
     * @param name name of the action to get
     * @returns {Observable<PipelineUsingAction>}
     */
    getUsage(name: string): Observable<Array<PipelineUsingAction>> {
        return this._http.get<Array<PipelineUsingAction>>('/action/' + name + '/usage');
    }

    /**
     * Create an action
     * @param action to create
     * @returns {Observable<Action>}
     */
    createAction(action: Action): Observable<Action> {
        return this._http.post<Action>('/action/' + action.name, action);
    }

    /**
     * Update an action
     * @param action to update
     * @returns {Observable<Action>}
     */
    updateAction(name: string, action: Action): Observable<Action> {
        return this._http.put<Action>('/action/' + name, action);
    }

    /**
     * Delete a action from CDS
     * @param name Actionname of the action to delete
     * @returns {Observable<Response>}
     */
    deleteAction(name: string): Observable<Response> {
        return this._http.delete<Response>('/action/' + name);
    }
}
