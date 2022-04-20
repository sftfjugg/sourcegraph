import * as React from 'react'

import styles from './CloseChangesetsListEmptyElement.module.scss'

export const CloseChangesetsListEmptyElement: React.FunctionComponent<{}> = () => (
    <div className={styles.closeChangesetsListEmptyElementBody}>
        <p className="text-center text-muted font-weight-normal">
            Closing this batch change will not alter changesets and no changesets will remain open.
        </p>
    </div>
)
